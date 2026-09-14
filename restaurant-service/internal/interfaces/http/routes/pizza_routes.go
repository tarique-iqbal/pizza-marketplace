package routes

import (
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupPizzaRoutes(router *gin.Engine, h *handlers.PizzaHandler, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.GET("/:id/pizzas", h.ListPizzas)
	protected.POST("/:id/pizzas", h.CreatePizza)
	protected.PUT("/:id/pizzas/:pizzaId", h.UpdatePizza)
	protected.PUT("/:id/pizzas/:pizzaId/prices", h.SetPizzaPrices)
}
