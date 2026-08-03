package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	AddressHandler  *handlers.AddressHandler
	DeliveryHandler *handlers.DeliveryHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers, m *middleware.Middleware) {
	SetupAddressRoutes(router, h.AddressHandler, m)
	SetupDeliveryRoutes(router, h.DeliveryHandler, m)
}
