package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupPayoutRoutes(router *gin.Engine, h *handlers.PayoutHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.POST("/:id/payout-details", h.CreatePayout)
	protected.PUT("/:id/payout-details", h.UpdatePayout)
}
