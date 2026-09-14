package routes

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

func SetupOrderRoutes(router *gin.Engine, h *handlers.OrderHandler, m *middleware.Middleware) {
	orders := router.Group("/orders")

	protected := orders.Group("")
	protected.Use(m.Auth)

	protected.POST("", h.Checkout)
	protected.GET("", h.ListMyOrders)
	protected.GET("/:id", h.GetOrder)
	protected.POST("/:id/cancel", h.Cancel)

	ownerOnly := orders.Group("")
	ownerOnly.Use(m.Auth, m.EnsureOwner)

	ownerOnly.GET("/restaurants/:id", h.ListRestaurantOrders)
	ownerOnly.POST("/:id/ready", h.MarkReady)
	ownerOnly.POST("/:id/complete", h.Complete)
}
