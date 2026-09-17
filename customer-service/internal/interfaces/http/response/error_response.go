package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	logobs "customer-service/internal/infrastructure/observability/logger"
	apperr "customer-service/internal/shared/errors"
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

	case errors.Is(err, apperr.ErrConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	default:
		logobs.FromContext(ctx.Request.Context()).Error("unhandled error", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
