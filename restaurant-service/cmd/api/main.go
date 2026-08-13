package main

import (
	"github.com/gin-gonic/gin"

	"restaurant-service/internal/container"
	"restaurant-service/internal/infrastructure/observability"
	logobs "restaurant-service/internal/infrastructure/observability/logger"
	"restaurant-service/internal/interfaces/http/routes"
)

func main() {
	logger := logobs.NewLogger("restaurant-api")

	app, err := container.NewAPIContainer()
	if err != nil {
		logger.Error("application exited with error", "error", err)
		return
	}
	defer app.Close()

	router := gin.New()
	router.Use(gin.Recovery(), observability.Middleware(logger))

	handlers := &routes.Handlers{
		AddressHandler:      app.AddressHandler,
		ContactHandler:      app.ContactHandler,
		DeliveryHandler:     app.DeliveryHandler,
		PayoutHandler:       app.PayoutHandler,
		OpeningHoursHandler: app.OpeningHoursHandler,
		ToppingPriceHandler: app.ToppingPriceHandler,
		PizzaHandler:        app.PizzaHandler,
	}

	routes.SetupRoutes(router, handlers, app.Middleware)

	if err := router.Run(":8080"); err != nil {
		logger.Error("failed to start server", "error", err)
	}
}
