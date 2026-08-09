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

type PizzaHandler struct {
	createPizza *commands.CreatePizza
}

func NewPizzaHandler(
	createPizza *commands.CreatePizza,
) *PizzaHandler {
	return &PizzaHandler{
		createPizza: createPizza,
	}
}

func (h *PizzaHandler) CreatePizza(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input resapp.CreatePizzaRequest

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

	res, err := h.createPizza.Execute(reqCtx, restaurantID, ownerID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, res)
}
