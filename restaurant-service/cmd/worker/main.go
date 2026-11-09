package main

import (
	"context"
	"os"
	"time"

	"restaurant-service/cmd/worker/bootstrap"

	"pizza-marketplace/shared/logobs"
	"pizza-marketplace/shared/telemetry"
)

const (
	serviceName           = "restaurant-worker"
	telemetryFlushTimeout = 5 * time.Second
)

func main() {
	logger := logobs.NewLogger(serviceName)

	shutdown, err := telemetry.Init(context.Background(), telemetry.Config{ServiceName: serviceName})
	if err != nil {
		logger.Error("failed to initialize telemetry", "error", err)
		os.Exit(1)
	}

	runErr := bootstrap.NewApp(logger).Run()

	flushCtx, cancel := context.WithTimeout(context.Background(), telemetryFlushTimeout)
	_ = shutdown(flushCtx)
	cancel()

	if runErr != nil {
		logger.Error("application exited with error", "error", runErr)
		os.Exit(1)
	}
}
