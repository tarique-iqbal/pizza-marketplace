package routes

import (
	"github.com/gin-gonic/gin"

	"search-service/internal/interfaces/http/handlers"
)

func SetupHealthRoutes(router *gin.Engine, handler *handlers.HealthHandler) {
	health := router.Group("/healthz")

	health.GET("/liveness", handler.Liveness)
	health.GET("/readiness", handler.Readiness)
}
