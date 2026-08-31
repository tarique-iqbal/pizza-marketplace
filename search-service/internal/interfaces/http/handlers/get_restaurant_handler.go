package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"search-service/internal/application/query"
	apperr "search-service/internal/shared/errors"
)

type GetRestaurantHandler struct {
	getRestaurant *query.GetRestaurant
}

func NewGetRestaurantHandler(getRestaurant *query.GetRestaurant) *GetRestaurantHandler {
	return &GetRestaurantHandler{getRestaurant: getRestaurant}
}

func (h *GetRestaurantHandler) Get(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid restaurant id"})
		return
	}

	restaurant, err := h.getRestaurant.Execute(ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "restaurant not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch restaurant"})
		return
	}

	ctx.JSON(http.StatusOK, restaurant)
}
