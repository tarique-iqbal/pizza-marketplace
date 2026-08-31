package main

import (
	"github.com/gin-gonic/gin"

	"search-service/internal/container"
	"search-service/internal/infrastructure/observability"
	logobs "search-service/internal/infrastructure/observability/logger"
	"search-service/internal/interfaces/http/routes"
)

func main() {
	logger := logobs.NewLogger("search-api")

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("application exited with error", "error", err)
		return
	}
	defer app.Close()

	router := gin.New()
	router.Use(gin.Recovery(), observability.Middleware(logger))

	handlers := &routes.Handlers{
		SearchHandler:        app.SearchHandler,
		GetRestaurantHandler: app.GetRestaurantHandler,
	}

	routes.SetupRoutes(router, handlers)

	if err := router.Run(":8080"); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
