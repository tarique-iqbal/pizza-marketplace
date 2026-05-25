package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"identity-service/internal/domain/auth"
)

const (
	CtxUserID   = "userID"
	CtxUserRole = "userRole"

	HeaderUserID   = "X-User-ID"
	HeaderUserRole = "X-User-Role"
)

func AuthMiddleware(jwtManager auth.JWTManager) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.GetHeader("Authorization")

		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			ctx.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		claims, err := jwtManager.Parse(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			ctx.Abort()
			return
		}

		ctx.Set(CtxUserID, claims.UserID)
		ctx.Set(CtxUserRole, claims.Role)

		ctx.Header(HeaderUserID, claims.UserID)
		ctx.Header(HeaderUserRole, claims.Role)

		ctx.Next()
	}
}
