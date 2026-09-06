package routes

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

func SetupAddItemRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth, m.EnsureCustomer)

	protected.POST("/items", h.AddItem)
}

func SetupUpdateItemQuantityRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth, m.EnsureCustomer)

	protected.PATCH("/items/:itemId", h.UpdateItemQuantity)
}

func SetupRemoveItemRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth, m.EnsureCustomer)

	protected.DELETE("/items/:itemId", h.RemoveItem)
}
