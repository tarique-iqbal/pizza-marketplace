package routes

import (
	"github.com/gin-gonic/gin"

	"customer-service/internal/interfaces/http/handlers"
	"customer-service/internal/interfaces/http/middleware"
)

func SetupAddressRoutes(router *gin.Engine, h *handlers.AddressHandler, m *middleware.Middleware) {
	addresses := router.Group("/customers/me/addresses")
	addresses.Use(m.Auth)

	addresses.GET("", h.List)
	addresses.DELETE("/:id", h.Delete)
	addresses.POST("/:id/default", h.SetDefault)
}
