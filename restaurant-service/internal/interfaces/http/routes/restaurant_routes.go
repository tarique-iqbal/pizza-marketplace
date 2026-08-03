package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAddressRoutes(router *gin.Engine, h *handlers.AddressHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PATCH("/:id/address", h.UpdateAddress)
}

func SetupDeliveryRoutes(router *gin.Engine, h *handlers.DeliveryHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PATCH("/:id/delivery", h.UpdateDelivery)
}
