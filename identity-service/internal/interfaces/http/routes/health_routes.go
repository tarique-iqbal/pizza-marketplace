package routes

import (
	"github.com/gin-gonic/gin"

	"identity-service/internal/interfaces/http"
)

func SetupHealthRoutes(router *gin.Engine, handler *http.HealthHandler) {
	health := router.Group("/health")

	health.GET("/live", handler.Live)
	health.GET("/ready", handler.Ready)
}
