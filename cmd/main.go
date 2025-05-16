package main

import (
	"context"
	"fmt"
	"go-dynamic-worker/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
)

type App struct {
	ctx context.Context
	cfg *config.Config
}

func SetupLogger() {
	w := os.Stderr
	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if err, ok := a.Value.Any().(error); ok {
					aErr := tint.Err(err)
					aErr.Key = a.Key
					return aErr
				}
				return a
			},
		}),
	)
	slog.SetDefault(logger)
}

func main() {
	SetupLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-signalCh:
			slog.Info("Received termination signal, shutting down", "signal", sig.String())
			cancel()
		case <-ctx.Done():
		}
	}()

	slog.Info("Running Test App")
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("Failed to load config")
		os.Exit(78)
	}
	app := App{
		ctx: ctx,
		cfg: cfg,
	}
	app.run()
}

func (app App) run() {
	ticker := time.NewTicker(time.Duration(1 * time.Second))
	for {
		select {
		case <-app.ctx.Done():
			slog.Info("Context cancelled, stopping main loop")
			return
		case <-ticker.C:
			fmt.Println("Running...")
		}
	}
}
