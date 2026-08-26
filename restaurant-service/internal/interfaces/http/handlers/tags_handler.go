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

type TagsHandler struct {
	updateTags *commands.UpdateTags
}

func NewTagsHandler(
	updateTags *commands.UpdateTags,
) *TagsHandler {
	return &TagsHandler{
		updateTags: updateTags,
	}
}

func (h *TagsHandler) UpdateTags(ctx *gin.Context) {
	reqCtx := ctx.Request.Context()

	var input resapp.UpdateTagsRequest

	idParam := ctx.Param("id")
	restaurantID, err := uuid.Parse(idParam)
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

	res, err := h.updateTags.Execute(reqCtx, restaurantID, ownerID, input)
	if err != nil {
		response.HandleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
