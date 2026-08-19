package main

import (
	"os"

	"search-service/cmd/worker/bootstrap"
	logobs "search-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("search-worker")

	if err := bootstrap.NewApp(logger).Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
