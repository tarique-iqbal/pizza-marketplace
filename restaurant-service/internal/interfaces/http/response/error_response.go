package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	apperr "restaurant-service/internal/shared/errors"

	"pizza-marketplace/shared/logobs"
)

func HandleError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, apperr.ErrUnauthorized):
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})

	case errors.Is(err, apperr.ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})

	case errors.Is(err, apperr.ErrNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, apperr.ErrInvalid):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

	case errors.Is(err, payout.ErrNoUnverifiedPayout):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no unverified payout submission to update"})

	case errors.Is(err, payout.ErrUnverifiedPayoutExists):
		ctx.JSON(http.StatusConflict, gin.H{"error": "a payout submission is already pending review"})

	case errors.Is(err, apperr.ErrConflict) ||
		errors.Is(err, restaurant.ErrEmailAlreadyExists):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	default:
		reqCtx := ctx.Request.Context()
		logobs.FromContext(reqCtx).ErrorContext(reqCtx, "unhandled error", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
