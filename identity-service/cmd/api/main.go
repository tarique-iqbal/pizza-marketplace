package main

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"identity-service/internal/container"
	"identity-service/internal/interfaces/http/routes"

	"pizza-marketplace/shared/logobs"
	"pizza-marketplace/shared/telemetry"
)

const (
	serviceName           = "identity-api"
	telemetryFlushTimeout = 5 * time.Second
)

func main() {
	logger := logobs.NewLogger(serviceName)

	shutdown, err := telemetry.Init(context.Background(), telemetry.Config{ServiceName: serviceName})
	if err != nil {
		logger.Error("failed to initialize telemetry", "error", err)
		return
	}
	defer flushTelemetry(shutdown)

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("failed to initialize API container", "error", err)
		return
	}
	defer app.Close()

	router := gin.New()
	router.Use(gin.Recovery(), telemetry.GinMiddleware(serviceName), logobs.Middleware(logger))

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

func flushTelemetry(shutdown telemetry.Shutdown) {
	ctx, cancel := context.WithTimeout(context.Background(), telemetryFlushTimeout)
	defer cancel()

	_ = shutdown(ctx)
}
