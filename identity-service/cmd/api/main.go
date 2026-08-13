package main

import (
	"github.com/gin-gonic/gin"

	"identity-service/internal/container"
	"identity-service/internal/infrastructure/observability"
	logobs "identity-service/internal/infrastructure/observability/logger"
	"identity-service/internal/interfaces/http/routes"
)

func main() {
	logger := logobs.NewLogger("identity-api")

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("failed to initialize API container", "error", err)
		return
	}
	defer app.Close()

	router := gin.New()
	router.Use(gin.Recovery(), observability.Middleware(logger))

	handlers := &routes.Handlers{
		UserHandler:   app.UserHandler,
		AuthHandler:   app.AuthHandler,
		HealthHandler: app.HealthHandler,
	}

	routes.SetupRoutes(router, handlers, app.Middleware)

	logger.Info("starting HTTP server", "addr", ":8080")

	if err := router.Run(":8080"); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
