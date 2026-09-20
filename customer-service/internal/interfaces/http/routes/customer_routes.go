package routes

import (
	"github.com/gin-gonic/gin"

	"customer-service/internal/interfaces/http/handlers"
	"customer-service/internal/interfaces/http/middleware"
)

func SetupCustomerRoutes(router *gin.Engine, h *handlers.CustomerHandler, m *middleware.Middleware) {
	customers := router.Group("/customers/me")
	customers.Use(m.Auth)

	customers.GET("", h.GetProfile)
	customers.PATCH("/phone", h.UpdatePhone)
}
