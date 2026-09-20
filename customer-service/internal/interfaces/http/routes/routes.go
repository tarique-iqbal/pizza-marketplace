package routes

import (
	"github.com/gin-gonic/gin"

	"customer-service/internal/interfaces/http/handlers"
	"customer-service/internal/interfaces/http/middleware"
)

type Handlers struct {
	CustomerHandler *handlers.CustomerHandler
	AddressHandler  *handlers.AddressHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers, m *middleware.Middleware) {
	SetupCustomerRoutes(router, h.CustomerHandler, m)
	SetupAddressRoutes(router, h.AddressHandler, m)
}
