package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	orderapp "order-service/internal/application/order"
	"order-service/internal/application/order/commands"
	"order-service/internal/interfaces/http/middleware"
	"order-service/internal/interfaces/http/response"
	"order-service/internal/interfaces/http/validation"
)

type OrderHandler struct {
	checkout *commands.Checkout
}

func NewOrderHandler(checkout *commands.Checkout) *OrderHandler {
	return &OrderHandler{checkout: checkout}
}

func (h *OrderHandler) Checkout(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input orderapp.CheckoutRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"errors": validation.ExtractValidationErrors(err),
		})
		return
	}

	userID := ctx.MustGet(middleware.CtxUserID).(string)

	customerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	res, err := h.checkout.Execute(reqCtx, customerID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, res)
}
