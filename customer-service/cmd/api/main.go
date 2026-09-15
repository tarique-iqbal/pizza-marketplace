package main

import (
	"github.com/gin-gonic/gin"

	"customer-service/internal/container"
	"customer-service/internal/infrastructure/observability"
	logobs "customer-service/internal/infrastructure/observability/logger"
)

func main() {
	logger := logobs.NewLogger("customer-api")

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
