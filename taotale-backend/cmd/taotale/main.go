package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/httpapi"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *listen); err != nil {
		slog.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, listen string) error {
	mediaRoot := os.Getenv("MEDIA_ROOT")
	if !filepath.IsAbs(mediaRoot) {
		return errors.New("MEDIA_ROOT must be an absolute path")
	}
	if err := os.MkdirAll(mediaRoot, 0700); err != nil {
		return errors.New("cannot create media directory")
	}
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return errors.New("invalid PostgreSQL configuration")
	}
	defer pool.Close()

	ready := func(ctx context.Context) error {
		if err := pool.Ping(ctx); err != nil {
			return err
		}
		file, err := os.CreateTemp(mediaRoot, ".health-*")
		if err != nil {
			return err
		}
		closeErr := file.Close()
		return errors.Join(closeErr, os.Remove(file.Name()))
	}
	server := &http.Server{
		Addr: listen, Handler: httpapi.Router(ready),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: time.Minute,
	}
	errs := make(chan error, 1)
	go func() { errs <- server.ListenAndServe() }()
	select {
	case err := <-errs:
		return fmt.Errorf("HTTP listener: %w", err)
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
