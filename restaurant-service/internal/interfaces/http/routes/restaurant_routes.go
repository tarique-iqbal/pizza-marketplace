package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAddressRoutes(router *gin.Engine, h *handlers.AddressHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PATCH("/:id/address", h.UpdateAddress)
}

func SetupContactRoutes(router *gin.Engine, h *handlers.ContactHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PATCH("/:id/contact", h.UpdateContact)
}

func SetupDeliveryRoutes(router *gin.Engine, h *handlers.DeliveryHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PATCH("/:id/delivery", h.UpdateDelivery)
}

func SetupPayoutRoutes(router *gin.Engine, h *handlers.PayoutHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.POST("/:id/payout-details", h.CreatePayout)
	protected.PUT("/:id/payout-details", h.UpdatePayout)
}

func SetupOpeningHoursRoutes(router *gin.Engine, h *handlers.OpeningHoursHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PATCH("/:id/opening-hours", h.UpdateOpeningHours)
}

func SetupToppingPriceRoutes(router *gin.Engine, h *handlers.ToppingPriceHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.PUT("/:id/topping-prices", h.SetToppingPrices)
}

func SetupPizzaRoutes(router *gin.Engine, h *handlers.PizzaHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.GET("/:id/pizzas", h.ListPizzas)
	protected.POST("/:id/pizzas", h.CreatePizza)
	protected.PUT("/:id/pizzas/:pizzaId", h.UpdatePizza)
	protected.PUT("/:id/pizzas/:pizzaId/prices", h.SetPizzaPrices)
}
