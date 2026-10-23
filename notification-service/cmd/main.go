package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"notification-service/internal/container"
	"notification-service/internal/infrastructure/messaging"
	logobs "notification-service/internal/infrastructure/observability/logger"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := logobs.NewLogger("notification-worker")

	app, err := container.NewContainer()
	if err != nil {
		logger.Error("failed to initialize container", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	ctx = logobs.WithContext(ctx, logger)

	var wg sync.WaitGroup

	logger.Info("starting consumer")

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer recoverPanic(stop, logger)

		if err := messaging.Run(ctx, app.Consumer, app.Dispatcher); err != nil &&
			!errors.Is(err, context.Canceled) {
			logger.Error("consumer stopped unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown initiated")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("shutdown complete")
	case <-time.After(shutdownTimeout):
		logger.Warn("forced shutdown after timeout")
	}
}

func recoverPanic(stop context.CancelFunc, logger *slog.Logger) {
	if rec := recover(); rec != nil {
		logger.Error(
			"worker panic",
			"panic", rec,
			"stack", string(debug.Stack()),
		)
		stop()
	}
}
