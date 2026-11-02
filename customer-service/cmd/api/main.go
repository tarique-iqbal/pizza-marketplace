package main

import (
	"github.com/gin-gonic/gin"

	"customer-service/internal/container"
	"customer-service/internal/interfaces/http/routes"

	"pizza-marketplace/shared/logobs"
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
	router.Use(gin.Recovery(), logobs.Middleware(logger))

	handlers := &routes.Handlers{
		CustomerHandler: app.CustomerHandler,
		AddressHandler:  app.AddressHandler,
		HealthHandler:   app.HealthHandler,
	}

	routes.SetupRoutes(router, handlers, app.Middleware)

	if err := router.Run(":8080"); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
