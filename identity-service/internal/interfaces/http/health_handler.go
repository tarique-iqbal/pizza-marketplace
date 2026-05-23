package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"identity-service/internal/application/health"
)

type HealthHandler struct {
	Readiness health.Checker
}

func NewHealthHandler(readiness health.Checker) *HealthHandler {
	return &HealthHandler{
		Readiness: readiness,
	}
}

func (h *HealthHandler) Live(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

func (h *HealthHandler) Ready(ctx *gin.Context) {
	c, cancel := context.WithTimeout(
		ctx.Request.Context(),
		2*time.Second,
	)
	defer cancel()

	results, healthy := h.Readiness.Check(c)

	response := gin.H{
		"status": "ready",
		"checks": results,
	}

	statusCode := http.StatusOK

	if !healthy {
		statusCode = http.StatusServiceUnavailable
		response["status"] = "not-ready"
	}

	ctx.JSON(statusCode, response)
}
