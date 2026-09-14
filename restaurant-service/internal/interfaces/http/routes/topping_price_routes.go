package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupToppingPriceRoutes(router *gin.Engine, h *handlers.ToppingPriceHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PUT("/:id/topping-prices", h.SetToppingPrices)
}
