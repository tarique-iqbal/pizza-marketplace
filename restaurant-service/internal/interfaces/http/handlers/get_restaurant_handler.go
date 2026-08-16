package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"restaurant-service/internal/application/restaurant/queries"
	"restaurant-service/internal/interfaces/http/response"
)

type GetRestaurantHandler struct {
	getRestaurant *queries.GetRestaurant
}

func NewGetRestaurantHandler(
	getRestaurant *queries.GetRestaurant,
) *GetRestaurantHandler {
	return &GetRestaurantHandler{
		getRestaurant: getRestaurant,
	}
}

func (h *GetRestaurantHandler) GetRestaurant(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	restaurantID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid restaurant id",
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

	res, err := h.getRestaurant.Execute(reqCtx, restaurantID, ownerID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
