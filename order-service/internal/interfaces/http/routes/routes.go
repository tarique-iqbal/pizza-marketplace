package routes

import (
	"github.com/gin-gonic/gin"

	"order-service/internal/interfaces/http/handlers"
	"order-service/internal/interfaces/http/middleware"
)

type Handlers struct {
	CartHandler  *handlers.CartHandler
	OrderHandler *handlers.OrderHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers, m *middleware.Middleware) {
	SetupCartRoutes(router, h.CartHandler, m)
	SetupOrderRoutes(router, h.OrderHandler, m)
}
