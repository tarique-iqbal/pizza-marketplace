package routes

import (
	"identity-service/internal/interfaces/http"
	"identity-service/internal/interfaces/http/middlewares"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	UserHandler   *http.UserHandler
	AuthHandler   *http.AuthHandler
	HealthHandler *http.HealthHandler
}

func SetupRoutes(router *gin.Engine, h *Handlers, m *middlewares.Middleware) {
	SetupUserRoutes(router, h.UserHandler, m)
	SetupAuthRoutes(router, h.AuthHandler, m)
	SetupHealthRoutes(router, h.HealthHandler)
}
