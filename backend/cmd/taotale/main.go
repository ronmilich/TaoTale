package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"taotale/backend/internal/app"
)

func main() {
	listen := flag.String("listen", app.DefaultListen, "local HTTP listen address")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	gin.SetMode(gin.ReleaseMode)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	config, err := app.LoadConfig(ctx, *listen, os.Getenv)
	if err == nil {
		err = app.Run(ctx, config, logger)
	}
	if err != nil {
		logger.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
