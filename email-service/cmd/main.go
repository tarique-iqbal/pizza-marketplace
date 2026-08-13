package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"email-service/internal/container"
	logobs "email-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("email-worker")

	app, err := container.NewContainer()
	if err != nil {
		logger.Error("failed to initialize container", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = logobs.WithContext(ctx, logger)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go app.Consumer.Run(ctx, app.Dispatcher)

	<-sigs
	logger.Info("shutdown signal received")
	cancel()
	app.Consumer.Close()
}
