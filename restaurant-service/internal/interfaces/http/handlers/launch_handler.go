package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/interfaces/http/response"
)

type LaunchHandler struct {
	launchRestaurant *commands.LaunchRestaurant
}

func NewLaunchHandler(
	launchRestaurant *commands.LaunchRestaurant,
) *LaunchHandler {
	return &LaunchHandler{
		launchRestaurant: launchRestaurant,
	}
}

func (h *LaunchHandler) Launch(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	idParam := ctx.Param("id")
	restaurantID, err := uuid.Parse(idParam)
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

	res, err := h.launchRestaurant.Execute(reqCtx, restaurantID, ownerID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
