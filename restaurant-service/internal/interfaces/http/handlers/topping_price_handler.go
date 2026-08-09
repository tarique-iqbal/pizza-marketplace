package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/interfaces/http/response"
	"restaurant-service/internal/interfaces/http/validation"
)

type ToppingPriceHandler struct {
	setToppingPrices *commands.SetToppingPrices
}

func NewToppingPriceHandler(setToppingPrices *commands.SetToppingPrices) *ToppingPriceHandler {
	return &ToppingPriceHandler{
		setToppingPrices: setToppingPrices,
	}
}

func (h *ToppingPriceHandler) SetToppingPrices(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input resapp.SetToppingPricesRequest

	restaurantID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid restaurant id",
		})
		return
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		validationErrors := validation.ExtractValidationErrors(err)

		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"errors": validationErrors,
		})
		return
	}

	userID := ctx.MustGet("userID").(string)

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid owner id",
		})
		return
	}

	res, err := h.setToppingPrices.Execute(reqCtx, restaurantID, ownerID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
