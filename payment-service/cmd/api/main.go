package main

import (
	"github.com/gin-gonic/gin"

	"payment-service/internal/container"
	"payment-service/internal/infrastructure/observability"
	logobs "payment-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("payment-api")

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("application exited with error", "error", err)
		return
	}
	defer app.Close()

	router := gin.New()
	router.Use(gin.Recovery(), observability.Middleware(logger))

	if err := router.Run(":8080"); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
