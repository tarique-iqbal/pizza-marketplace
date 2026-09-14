package routes

import (
	"restaurant-service/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRestaurantRoutes(router *gin.Engine, h *Handlers, m *middleware.Middleware) {
	restaurants := router.Group("/restaurants")

	protected := restaurants.Group("")
	protected.Use(m.Auth, m.EnsureOwner)

	protected.GET("/:id", h.GetRestaurantHandler.GetRestaurant)
	protected.PATCH("/:id/address", h.AddressHandler.UpdateAddress)
	protected.PATCH("/:id/contact", h.ContactHandler.UpdateContact)
	protected.PATCH("/:id/delivery", h.DeliveryHandler.UpdateDelivery)
	protected.PATCH("/:id/tags", h.TagsHandler.UpdateTags)
	protected.PATCH("/:id/opening-hours", h.OpeningHoursHandler.UpdateOpeningHours)
	protected.POST("/:id/launch", h.LaunchHandler.Launch)
	protected.GET("/:id/launch", h.LaunchHandler.LaunchReadiness)

	admin := restaurants.Group("")
	admin.Use(m.Auth, m.EnsureAdmin)

	admin.POST("/:id/approve", h.ApproveHandler.Approve)
}
