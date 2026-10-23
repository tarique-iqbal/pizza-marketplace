package main

import (
	"os"

	"identity-service/cmd/worker/bootstrap"
	logobs "identity-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("identity-worker")

	if err := bootstrap.NewApp(logger).Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
