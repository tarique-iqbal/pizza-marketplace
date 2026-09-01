package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/container"
	"notification-service/internal/infrastructure/messaging"
	logobs "notification-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("notification-worker")

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

	go messaging.Run(ctx, app.Consumer, app.Dispatcher)

	<-sigs
	logger.Info("shutdown signal received")
	cancel()
	app.Consumer.Close()
}
