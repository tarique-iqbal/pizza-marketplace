package routes

import (
	"github.com/gin-gonic/gin"

	"identity-service/internal/interfaces/http"
	"identity-service/internal/interfaces/http/middlewares"
)

func SetupAuthRoutes(router *gin.Engine, handler *http.AuthHandler, m *middlewares.Middleware) {
	auth := router.Group("/auth")

	auth.POST("/email/verify", handler.CreateEmailVerification)
	auth.POST("/login", handler.Login)
	auth.POST("/refresh", handler.Refresh)

	protected := auth.Group("")
	protected.Use(m.Auth)

	protected.GET("/verify", handler.Verify)
	protected.POST("/logout", handler.Logout)
}
