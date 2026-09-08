package routes

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

func SetupAddItemRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth)

	protected.POST("/items", h.AddItem)
}

func SetupUpdateItemQuantityRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth)

	protected.PATCH("/items/:itemId", h.UpdateItemQuantity)
}

func SetupRemoveItemRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth)

	protected.DELETE("/items/:itemId", h.RemoveItem)
}

func SetupGetCartRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth)

	protected.GET("", h.GetCart)
}
