package main

import (
	"os"

	"restaurant-service/cmd/worker/bootstrap"
	logobs "restaurant-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("restaurant-worker")

	if err := bootstrap.NewApp(logger).Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
