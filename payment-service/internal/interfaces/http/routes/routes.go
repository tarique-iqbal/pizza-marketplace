package routes

import (
	"github.com/gin-gonic/gin"

	"payment-service/internal/interfaces/http/handlers"
)

type Handlers struct {
	WebhookHandler *handlers.WebhookHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers) {
	router.POST("/webhooks/mollie", h.WebhookHandler.HandleMollie)
}
