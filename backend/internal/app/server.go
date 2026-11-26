package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"taotale/backend/internal/httpapi"
	"taotale/backend/internal/media"
	"taotale/backend/internal/persistence"
	"taotale/backend/internal/persistence/queries"
)

func Run(ctx context.Context, config Config, logger *slog.Logger) error {
	if err := media.PrepareRoot(config.MediaRoot); err != nil {
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config.Database)
	if err != nil {
		return errors.New("cannot initialize PostgreSQL pool")
	}
	defer pool.Close()

	schema, err := persistence.LoadSchema()
	if err != nil {
		return errors.New("cannot load embedded database migrations")
	}
	if err := schema.Prepare(ctx, pool); err != nil {
		if errors.Is(err, persistence.ErrSchemaHistoryMismatch) {
			return err
		}
		return errors.New("cannot migrate and seed PostgreSQL")
	}

	ready := func(ctx context.Context) error {
		if err := schema.Check(ctx, pool); err != nil {
			return err
		}
		file, err := os.CreateTemp(config.MediaRoot, ".health-*")
		if err != nil {
			return err
		}
		return errors.Join(file.Close(), os.Remove(file.Name()))
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", config.Listen)
	if err != nil {
		return errors.New("cannot open HTTP listener")
	}
	server := &http.Server{
		Handler:           httpapi.Router(ready, logger, config.DevOrigin, queries.New(pool)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    16 * 1024,
	}
	return serve(ctx, listener, server)
}

func serve(ctx context.Context, listener net.Listener, server *http.Server) error {
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return errors.New("HTTP listener stopped unexpectedly")
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), httpapi.RequestTimeout)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			<-result
			return errors.New("HTTP shutdown did not complete gracefully")
		}
		<-result
		return nil
	}
}
