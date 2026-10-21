package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"restaurant-service/internal/interfaces/http/handlers"

	"pizza-marketplace/shared/health"
)

type mockChecker struct {
	checkFn func(ctx context.Context) (
		[]health.Result,
		bool,
	)
}

func (m mockChecker) Check(
	ctx context.Context,
) (
	[]health.Result,
	bool,
) {
	return m.checkFn(ctx)
}

func setupHealthRouter(handler *handlers.HealthHandler) *gin.Engine {
	r := gin.New()

	r.GET("/healthz/liveness", handler.Liveness)
	r.GET("/healthz/readiness", handler.Readiness)

	return r
}

func TestHealthHandler_Liveness(t *testing.T) {
	handler := handlers.NewHealthHandler(nil)

	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz/liveness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"alive"`)
}

func TestHealthHandler_Readiness_Healthy(t *testing.T) {
	checks := []health.Result{
		{Name: "postgres", Status: "up"},
	}

	handler := handlers.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				return checks, true
			},
		},
	)

	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz/readiness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "ready", body["status"])
	assert.NotNil(t, body["checks"])
}

func TestHealthHandler_Readiness_Unhealthy(t *testing.T) {
	checks := []health.Result{
		{Name: "postgres", Status: "down", Error: "connection refused"},
	}

	handler := handlers.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				return checks, false
			},
		},
	)

	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz/readiness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "not-ready", body["status"])
	assert.NotNil(t, body["checks"])
}

func TestHealthHandler_Readiness_PassesTimeoutContext(t *testing.T) {
	handler := handlers.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				deadline, ok := ctx.Deadline()
				assert.True(t, ok)

				remaining := time.Until(deadline)

				assert.LessOrEqual(t, remaining, 2*time.Second)
				assert.Greater(t, remaining, time.Second)

				return []health.Result{{Name: "postgres", Status: "up"}}, true
			},
		},
	)

	router := setupHealthRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz/readiness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
