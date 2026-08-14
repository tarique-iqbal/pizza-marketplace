package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/interfaces/http/response"
)

type ApproveHandler struct {
	approveRestaurant *commands.ApproveRestaurant
}

func NewApproveHandler(
	approveRestaurant *commands.ApproveRestaurant,
) *ApproveHandler {
	return &ApproveHandler{
		approveRestaurant: approveRestaurant,
	}
}

func (h *ApproveHandler) Approve(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	idParam := ctx.Param("id")
	restaurantID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid restaurant id",
		})
		return
	}

	res, err := h.approveRestaurant.Execute(reqCtx, restaurantID)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
