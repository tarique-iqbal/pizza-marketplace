package routes

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

func SetupCheckoutRoutes(router *gin.Engine, h *handlers.OrderHandler, m *middleware.Middleware) {
	orders := router.Group("/orders")

	protected := orders.Group("")
	protected.Use(m.Auth)

	protected.POST("", h.Checkout)
}

func SetupGetOrderRoutes(router *gin.Engine, h *handlers.OrderHandler, m *middleware.Middleware) {
	orders := router.Group("/orders")

	protected := orders.Group("")
	protected.Use(m.Auth)

	protected.GET("/:id", h.GetOrder)
}

func SetupListMyOrdersRoutes(router *gin.Engine, h *handlers.OrderHandler, m *middleware.Middleware) {
	orders := router.Group("/orders")

	protected := orders.Group("")
	protected.Use(m.Auth)

	protected.GET("", h.ListMyOrders)
}

func SetupListRestaurantOrdersRoutes(router *gin.Engine, h *handlers.OrderHandler, m *middleware.Middleware) {
	orders := router.Group("/orders")

	ownerOnly := orders.Group("")
	ownerOnly.Use(m.Auth, m.EnsureOwner)

	ownerOnly.GET("/restaurants/:id", h.ListRestaurantOrders)
}
