package routes

import (
	"github.com/gin-gonic/gin"

	"identity-service/internal/interfaces/http"
)

func SetupHealthRoutes(router *gin.Engine, handler *http.HealthHandler) {
	health := router.Group("/healthz")

	health.GET("/liveness", handler.Liveness)
	health.GET("/readiness", handler.Readiness)
}
