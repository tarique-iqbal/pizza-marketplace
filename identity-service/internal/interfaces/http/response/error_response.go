package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/user"
	logobs "identity-service/internal/infrastructure/observability/logger"
	apperr "identity-service/internal/shared/errors"
)

func HandleError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, apperr.ErrUnauthorized):
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})

	case errors.Is(err, apperr.ErrForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})

	case errors.Is(err, apperr.ErrNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, auth.ErrCodeInvalid) || errors.Is(err, auth.ErrCodeNotIssued):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, auth.ErrCodeExpired):
		ctx.JSON(http.StatusGone, gin.H{"error": err.Error()})

	case errors.Is(err, auth.ErrCodeUsed) ||
		errors.Is(err, apperr.ErrConflict) ||
		errors.Is(err, user.ErrEmailAlreadyExists):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	default:
		logobs.FromContext(ctx.Request.Context()).Error("unhandled error", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
