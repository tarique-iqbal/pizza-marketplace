package main

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"identity-service/internal/container"
	"identity-service/internal/interfaces/http/routes"
	"identity-service/internal/logger"
)

func main() {
	l := logger.New()
	slog.SetDefault(l)

	app, err := container.NewAPIContainer()
	if err != nil {
		slog.Error("failed to initialize API container", "error", err)
		return
	}
	defer app.Close()

	router := gin.Default()

	handlers := &routes.Handlers{
		UserHandler:   app.UserHandler,
		AuthHandler:   app.AuthHandler,
		HealthHandler: app.HealthHandler,
	}

	routes.SetupRoutes(router, handlers, app.Middleware)

	slog.Info("starting HTTP server", "addr", ":8080")

	if err := router.Run(":8080"); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
