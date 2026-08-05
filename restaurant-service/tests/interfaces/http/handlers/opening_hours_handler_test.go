package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

type openingHoursHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.OpeningHoursHandler
}

func setupOpeningHoursHandler(t *testing.T) openingHoursHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	repo := persistence.NewRestaurantRepository(db.DB)
	updateOpeningHours := commands.NewUpdateOpeningHours(repo)
	handler := handlers.NewOpeningHoursHandler(updateOpeningHours)

	return openingHoursHandlerSetup{
		DB:      db.DB,
		Handler: handler,
	}
}

func TestOpeningHoursHandler_UpdateOpeningHours_Success(t *testing.T) {
	h := setupOpeningHoursHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/opening-hours", h.Handler.UpdateOpeningHours)

	reqBody := map[string]any{
		"monday":    []map[string]string{{"open": "11:00", "close": "22:00"}},
		"tuesday":   []map[string]string{{"open": "11:00", "close": "22:00"}},
		"wednesday": []map[string]string{{"open": "11:00", "close": "22:00"}},
		"thursday":  []map[string]string{{"open": "11:00", "close": "22:00"}},
		"friday":    []map[string]string{{"open": "11:00", "close": "23:00"}},
		"saturday":  []map[string]string{{"open": "00:00", "close": "03:00"}, {"open": "12:00", "close": "23:59"}},
		"sunday":    []map[string]string{},
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/opening-hours",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response resapp.RestaurantResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, res.ID.String(), response.ID.String())
	assert.Equal(t, "11:00", response.OpeningHours.Monday[0].Open)
	require.Len(t, response.OpeningHours.Saturday, 2)
	assert.Equal(t, "00:00", response.OpeningHours.Saturday[0].Open)
	assert.Empty(t, response.OpeningHours.Sunday)

	var updated restaurant.Restaurant
	err = h.DB.First(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistOpeningHours])
}

func TestOpeningHoursHandler_UpdateOpeningHours_Failure_InvalidFormat(t *testing.T) {
	h := setupOpeningHoursHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/opening-hours", h.Handler.UpdateOpeningHours)

	payload := `{"monday": [{"open": "9am", "close": "22:00"}]}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/opening-hours",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestOpeningHoursHandler_UpdateOpeningHours_Failure_CloseBeforeOpen(t *testing.T) {
	h := setupOpeningHoursHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/opening-hours", h.Handler.UpdateOpeningHours)

	payload := `{"monday": [{"open": "22:00", "close": "11:00"}]}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/opening-hours",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestOpeningHoursHandler_UpdateOpeningHours_Failure_Unauthorized(t *testing.T) {
	h := setupOpeningHoursHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))

	router.PATCH("/restaurants/:id/opening-hours", h.Handler.UpdateOpeningHours)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/opening-hours",
		bytes.NewBufferString(`{"monday": [{"open": "11:00", "close": "22:00"}]}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing X-User-ID header")
}

func TestOpeningHoursHandler_UpdateOpeningHours_Failure_Forbidden_WrongRole(t *testing.T) {
	h := setupOpeningHoursHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "customer"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/opening-hours", h.Handler.UpdateOpeningHours)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/opening-hours",
		bytes.NewBufferString(`{"monday": [{"open": "11:00", "close": "22:00"}]}`),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "access denied")
}
