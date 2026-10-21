package http_test

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

	httpui "identity-service/internal/interfaces/http"
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

func setupRouter(handler *httpui.HealthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()

	r.GET("/healthz/liveness", handler.Liveness)
	r.GET("/healthz/readiness", handler.Readiness)

	return r
}

func TestHealthHandler_Liveness(t *testing.T) {
	handler := httpui.NewHealthHandler(nil)

	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz/liveness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"alive"`)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestHealthHandler_Readiness_Healthy(t *testing.T) {
	checks := []health.Result{
		{Name: "db", Status: "ok"},
		{Name: "redis", Status: "ok"},
	}

	handler := httpui.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				return checks, true
			},
		},
	)

	router := setupRouter(handler)

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
		{Name: "db", Status: "ok"},
		{Name: "redis", Status: "down", Error: "connection refused"},
	}

	handler := httpui.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				return checks, false
			},
		},
	)

	router := setupRouter(handler)

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
	handler := httpui.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				deadline, ok := ctx.Deadline()
				assert.True(t, ok)

				remaining := time.Until(deadline)

				assert.LessOrEqual(t, remaining, 2*time.Second)
				assert.Greater(t, remaining, time.Second)

				return []health.Result{{Name: "db", Status: "ok"}}, true
			},
		},
	)

	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz/readiness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealthHandler_Readiness_EmptyChecks(t *testing.T) {
	handler := httpui.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				return []health.Result{}, true
			},
		},
	)

	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/healthz/readiness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"checks":[]`)
}
