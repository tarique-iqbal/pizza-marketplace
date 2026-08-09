package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	AddressHandler      *handlers.AddressHandler
	ContactHandler      *handlers.ContactHandler
	DeliveryHandler     *handlers.DeliveryHandler
	PayoutHandler       *handlers.PayoutHandler
	OpeningHoursHandler *handlers.OpeningHoursHandler
	ToppingPriceHandler *handlers.ToppingPriceHandler
	PizzaHandler        *handlers.PizzaHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers, m *middleware.Middleware) {
	SetupAddressRoutes(router, h.AddressHandler, m)
	SetupContactRoutes(router, h.ContactHandler, m)
	SetupDeliveryRoutes(router, h.DeliveryHandler, m)
	SetupPayoutRoutes(router, h.PayoutHandler, m)
	SetupOpeningHoursRoutes(router, h.OpeningHoursHandler, m)
	SetupToppingPriceRoutes(router, h.ToppingPriceHandler, m)
	SetupPizzaRoutes(router, h.PizzaHandler, m)
}
