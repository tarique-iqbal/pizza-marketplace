package main

import (
	"os"

	"notification-service/cmd/worker/bootstrap"
	logobs "notification-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("notification-worker")

	if err := bootstrap.NewApp(logger).Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
