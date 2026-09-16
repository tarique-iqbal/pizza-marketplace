package main

import (
	"os"

	"customer-service/cmd/worker/bootstrap"
	logobs "customer-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("customer-worker")

	if err := bootstrap.NewApp(logger).Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
