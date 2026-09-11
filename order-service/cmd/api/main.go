package main

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/container"
	"order-service/internal/infrastructure/observability"
	logobs "order-service/internal/infrastructure/observability/logger"
	"order-service/internal/interfaces/http/routes"
)

func main() {
	logger := logobs.NewLogger("order-api")

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("application exited with error", "error", err)
		return
	}
	defer app.Close()

	router := gin.New()
	router.Use(gin.Recovery(), observability.Middleware(logger))

	handlers := &routes.Handlers{
		CartHandler:  app.CartHandler,
		OrderHandler: app.OrderHandler,
	}

	routes.SetupRoutes(router, handlers, app.Middleware)

	if err := router.Run(":8080"); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
