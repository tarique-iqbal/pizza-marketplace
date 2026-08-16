package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	GetRestaurantHandler *handlers.GetRestaurantHandler
	AddressHandler       *handlers.AddressHandler
	ContactHandler       *handlers.ContactHandler
	DeliveryHandler      *handlers.DeliveryHandler
	PayoutHandler        *handlers.PayoutHandler
	OpeningHoursHandler  *handlers.OpeningHoursHandler
	ToppingPriceHandler  *handlers.ToppingPriceHandler
	PizzaHandler         *handlers.PizzaHandler
	ApproveHandler       *handlers.ApproveHandler
	LaunchHandler        *handlers.LaunchHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers, m *middleware.Middleware) {
	SetupGetRestaurantRoutes(router, h.GetRestaurantHandler, m)
	SetupAddressRoutes(router, h.AddressHandler, m)
	SetupContactRoutes(router, h.ContactHandler, m)
	SetupDeliveryRoutes(router, h.DeliveryHandler, m)
	SetupPayoutRoutes(router, h.PayoutHandler, m)
	SetupOpeningHoursRoutes(router, h.OpeningHoursHandler, m)
	SetupToppingPriceRoutes(router, h.ToppingPriceHandler, m)
	SetupPizzaRoutes(router, h.PizzaHandler, m)
	SetupApproveRoutes(router, h.ApproveHandler, m)
	SetupLaunchRoutes(router, h.LaunchHandler, m)
}
