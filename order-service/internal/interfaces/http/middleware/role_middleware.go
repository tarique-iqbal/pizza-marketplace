package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRole(expectedRole string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userRole := ctx.MustGet(CtxUserRole).(string)

		if userRole != expectedRole {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "access denied",
			})
			return
		}

		ctx.Next()
	}
}
