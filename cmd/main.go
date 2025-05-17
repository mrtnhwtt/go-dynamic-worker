package main

import (
	"context"
	"go-dynamic-worker/internal/config"
	"go-dynamic-worker/internal/queue"
	"go-dynamic-worker/internal/sender"
	"go-dynamic-worker/internal/worker"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
)

type App struct {
	ctx        context.Context
	cfg        *config.Config
	queue      queue.QueueService
	workerPool worker.WorkerPoolService
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

	q, err := queue.NewQueue(0*time.Second, 0.99)
	if err != nil {
		slog.Error("failed to create a queue service")
	}

	ss := sender.NewSender()
	workerPool := worker.NewWorkerPool(ctx, 2, ss, q)
	if err := workerPool.Init(); err != nil {
		slog.Error("failed to initialize worker pool", "error", err)
	}

	app := App{
		ctx:        ctx,
		cfg:        cfg,
		queue:      q,
		workerPool: workerPool,
	}
	app.run()
}

func (app App) run() {
	// ticker := time.NewTicker(time.Duration(1 * time.Second))
	// defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			slog.Info("Context cancelled, stopping main loop")
			app.workerPool.Stop()
			return
		default:
			messages, err := app.queue.PollQueue(app.cfg.WorkerCount)
			if err != nil {
				slog.Error("failed to retrieve message", "error", err)
				continue
			}
			if len(messages) == 0 {
				slog.Warn("No message received")
				continue
			}
			slog.Info("Received Message, submitting...", "messageCount", len(messages))
			app.workerPool.SubmitMessage(messages) // Blocking operation if no worker available
			slog.Info("Finished submitting.")
		}
	}
}
