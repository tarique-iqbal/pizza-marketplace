package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pizza-marketplace/shared/health"
)

type HealthHandler struct {
	Checker health.Checker
}

func NewHealthHandler(checker health.Checker) *HealthHandler {
	return &HealthHandler{
		Checker: checker,
	}
}

func (h *HealthHandler) Liveness(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

func (h *HealthHandler) Readiness(ctx *gin.Context) {
	c, cancel := context.WithTimeout(
		ctx.Request.Context(),
		2*time.Second,
	)
	defer cancel()

	results, healthy := h.Checker.Check(c)

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
