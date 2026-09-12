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
	SetupAddItemRoutes(router, h.CartHandler, m)
	SetupUpdateItemQuantityRoutes(router, h.CartHandler, m)
	SetupRemoveItemRoutes(router, h.CartHandler, m)
	SetupGetCartRoutes(router, h.CartHandler, m)
	SetupCheckoutRoutes(router, h.OrderHandler, m)
	SetupGetOrderRoutes(router, h.OrderHandler, m)
	SetupListMyOrdersRoutes(router, h.OrderHandler, m)
	SetupListRestaurantOrdersRoutes(router, h.OrderHandler, m)
}
