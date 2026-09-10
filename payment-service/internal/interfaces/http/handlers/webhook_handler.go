package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payment-service/internal/application/payment/commands"
	logobs "payment-service/internal/infrastructure/observability/logger"
)

type WebhookHandler struct {
	handleMollieWebhook *commands.HandleMollieWebhook
}

func NewWebhookHandler(handleMollieWebhook *commands.HandleMollieWebhook) *WebhookHandler {
	return &WebhookHandler{handleMollieWebhook: handleMollieWebhook}
}

func (h *WebhookHandler) HandleMollie(c *gin.Context) {
	id := c.PostForm("id")
	if id == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if err := h.handleMollieWebhook.Execute(c.Request.Context(), id); err != nil {
		logobs.FromContext(c.Request.Context()).Error(
			"failed to handle mollie webhook",
			"gateway_payment_id", id,
			"error", err,
		)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}
