package main

import (
	"os"

	"payment-service/cmd/worker/bootstrap"
	logobs "payment-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("payment-worker")

	if err := bootstrap.NewApp(logger).Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
