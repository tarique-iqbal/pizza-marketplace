package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"identity-service/internal/container"
	logobs "identity-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("identity-worker")

	c, err := container.NewWorkerContainer(logger)
	if err != nil {
		logger.Error("failed to initialize worker container", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	var wg sync.WaitGroup

	logger.Info("starting outbox worker...")

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Worker.Start(ctx)
	}()

	<-ctx.Done()

	logger.Info("shutdown signal received")

	wg.Wait()

	logger.Info("worker stopped gracefully")
}
