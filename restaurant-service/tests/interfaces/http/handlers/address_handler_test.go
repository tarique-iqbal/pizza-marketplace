package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/application/restaurant/commands"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type mockGeocoder struct {
	lat      float64
	lon      float64
	timezone string
	err      error
}

func (m *mockGeocoder) GeocodeAddress(
	ctx context.Context,
	addr restaurant.Address,
) (float64, float64, string, error) {
	return m.lat, m.lon, m.timezone, m.err
}

type addressHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.AddressHandler
}

func setupAddressHandler(t *testing.T) addressHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	mockGeo := &mockGeocoder{lat: 52.52, lon: 13.405, timezone: "Europe/Berlin", err: nil}
	repo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	updateAddress := commands.NewUpdateAddress(db.DB, mockGeo, repo, payoutDetailsRepo, outboxRepo)
	handler := handlers.NewAddressHandler(updateAddress)

	return addressHandlerSetup{
		DB:      db.DB,
		Handler: handler,
	}
}

func TestAddressHandler_UpdateAddress_Success(t *testing.T) {
	h := setupAddressHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/address", h.Handler.UpdateAddress)

	reqBody := map[string]any{
		"house":      "1",
		"street":     "Main Str.",
		"city":       "Cityville",
		"postalCode": "12345",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/address",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response resapp.RestaurantResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.Equal(t, res.ID.String(), response.ID.String())

	assert.NotEmpty(t, response.Slug)
	assert.Contains(t, *response.Slug, "cityville")

	assert.Equal(t, "1", response.Address.House)
	assert.Equal(t, "Main Str.", response.Address.Street)
	assert.Equal(t, "Cityville", response.Address.City)
	assert.Equal(t, "12345", response.Address.PostalCode)

	assert.NotZero(t, response.Lat)
	assert.NotZero(t, response.Lon)

	var updated restaurant.Restaurant
	err = h.DB.First(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.NotEmpty(t, updated.Slug)
	assert.Equal(t, "1", updated.Address.House)
	assert.Equal(t, "Main Str.", updated.Address.Street)
	assert.Equal(t, "Cityville", updated.Address.City)
	assert.Equal(t, "12345", updated.Address.PostalCode)
}

func TestAddressHandler_UpdateAddress_Failure_ValidationError(t *testing.T) {
	h := setupAddressHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/address", h.Handler.UpdateAddress)

	payload := `{"city": "Cityville"}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/address",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestAddressHandler_UpdateAddress_Failure_Unauthorized(t *testing.T) {
	h := setupAddressHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	// use the real AuthMiddleware
	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))

	router.PATCH("/restaurants/:id/address", h.Handler.UpdateAddress)

	reqBody := map[string]any{
		"house":      "1",
		"street":     "Main Str.",
		"city":       "Cityville",
		"postalCode": "12345",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/address",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")
	// no "X-User-ID" or "X-User-Role" headers

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing X-User-ID header")
}

func TestAddressHandler_UpdateAddress_Failure_Forbidden_MissingRole(t *testing.T) {
	h := setupAddressHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware("", ""),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/address", h.Handler.UpdateAddress)

	reqBody := map[string]any{
		"house":      "1",
		"street":     "Main Str.",
		"city":       "Cityville",
		"postalCode": "12345",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/address",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "access denied")
}

func TestAddressHandler_UpdateAddress_Failure_Forbidden_WrongRole(t *testing.T) {
	h := setupAddressHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	// wrong role injected
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "customer"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/address", h.Handler.UpdateAddress)

	reqBody := map[string]any{
		"house":      "1",
		"street":     "Main Str.",
		"city":       "Cityville",
		"postalCode": "12345",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/address",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "access denied")
}
