package routes

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

func SetupCheckoutRoutes(router *gin.Engine, h *handlers.OrderHandler, m *middleware.Middleware) {
	orders := router.Group("/orders")

	protected := orders.Group("")
	protected.Use(m.Auth, m.EnsureCustomer)

	protected.POST("", h.Checkout)
}
