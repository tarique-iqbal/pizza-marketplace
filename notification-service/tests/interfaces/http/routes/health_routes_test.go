package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"notification-service/internal/interfaces/http/handlers"
	"notification-service/internal/interfaces/http/routes"

	"pizza-marketplace/shared/health"
)

type mockChecker struct {
	healthy bool
}

var _ health.Checker = (*mockChecker)(nil)

func (m mockChecker) Check(ctx context.Context) ([]health.Result, bool) {
	return []health.Result{{Name: "rabbitmq", Status: "up"}}, m.healthy
}

func setupHealthRouter(checker health.Checker) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	handler := handlers.NewHealthHandler(checker)

	routes.SetupHealthRoutes(r, handler)

	return r
}

func TestSetupHealthRoutes_Liveness(t *testing.T) {
	router := setupHealthRouter(mockChecker{healthy: true})

	req := httptest.NewRequest(http.MethodGet, "/healthz/liveness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"alive"`)
}

func TestSetupHealthRoutes_Readiness_Healthy(t *testing.T) {
	router := setupHealthRouter(mockChecker{healthy: true})

	req := httptest.NewRequest(http.MethodGet, "/healthz/readiness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ready"`)
}

func TestSetupHealthRoutes_Readiness_Unhealthy(t *testing.T) {
	router := setupHealthRouter(mockChecker{healthy: false})

	req := httptest.NewRequest(http.MethodGet, "/healthz/readiness", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"not-ready"`)
}
