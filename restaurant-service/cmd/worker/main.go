package main

import (
	"os"

	"restaurant-service/cmd/worker/bootstrap"

	"pizza-marketplace/shared/logobs"
)

func main() {
	logger := logobs.NewLogger("restaurant-worker")

	if err := bootstrap.NewApp(logger).Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
