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

	"identity-service/internal/application/health"
	httpui "identity-service/internal/interfaces/http"
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

	r.GET("/live", handler.Live)
	r.GET("/ready", handler.Ready)

	return r
}

func TestHealthHandler_Live(t *testing.T) {
	handler := httpui.NewHealthHandler(nil)

	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"alive"`)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
}

func TestHealthHandler_Ready_Healthy(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "ready", body["status"])
	assert.NotNil(t, body["checks"])
}

func TestHealthHandler_Ready_Unhealthy(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "not-ready", body["status"])
	assert.NotNil(t, body["checks"])
}

func TestHealthHandler_Ready_PassesTimeoutContext(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealthHandler_Ready_EmptyChecks(t *testing.T) {
	handler := httpui.NewHealthHandler(
		mockChecker{
			checkFn: func(ctx context.Context) ([]health.Result, bool) {
				return []health.Result{}, true
			},
		},
	)

	router := setupRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"checks":[]`)
}
