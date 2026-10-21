package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_Liveness(t *testing.T) {
	router := newRouter(t)

	recorder := doRequest(router, http.MethodGet, "/healthz/liveness", nil, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"status":"alive"`)
}

func TestHealthHandler_Readiness(t *testing.T) {
	router := newRouter(t)

	recorder := doRequest(router, http.MethodGet, "/healthz/readiness", nil, nil)

	require.Equal(t, http.StatusOK, recorder.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))

	assert.Equal(t, "ready", body["status"])

	checks, ok := body["checks"].([]any)
	require.True(t, ok)
	require.Len(t, checks, 1)

	check := checks[0].(map[string]any)
	assert.Equal(t, "postgres", check["name"])
	assert.Equal(t, "up", check["status"])
}
