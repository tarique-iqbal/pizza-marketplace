package routes

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

func SetupCartRoutes(router *gin.Engine, h *handlers.CartHandler, m *middleware.Middleware) {
	cart := router.Group("/cart")

	protected := cart.Group("")
	protected.Use(m.Auth)

	protected.GET("", h.GetCart)
	protected.POST("/items", h.AddItem)
	protected.PATCH("/items/:itemId", h.UpdateItemQuantity)
	protected.DELETE("/items/:itemId", h.RemoveItem)
}
