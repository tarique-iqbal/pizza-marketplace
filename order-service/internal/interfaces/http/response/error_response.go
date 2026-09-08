package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"order-service/internal/domain/cart"
	"order-service/internal/domain/order"
	logobs "order-service/internal/infrastructure/observability/logger"
	apperr "order-service/internal/shared/errors"
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

	case errors.Is(err, cart.ErrCartRestaurantMismatch):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, cart.ErrCartEmpty):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

	case errors.Is(err, cart.ErrCartItemUnavailable):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, order.ErrOutsideDeliveryRadius):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

	case errors.Is(err, order.ErrGeocodingUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})

	case errors.Is(err, order.ErrFulfillmentNotSupported):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, order.ErrBelowMinimumOrder):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})

	case errors.Is(err, apperr.ErrConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	default:
		logobs.FromContext(ctx.Request.Context()).Error("unhandled error", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
