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

type deliveryHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.DeliveryHandler
}

func setupDeliveryHandler(t *testing.T) deliveryHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	repo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	updateDelivery := commands.NewUpdateDelivery(db.DB, repo, payoutDetailsRepo, outboxRepo)
	handler := handlers.NewDeliveryHandler(updateDelivery)

	return deliveryHandlerSetup{
		DB:      db.DB,
		Handler: handler,
	}
}

func TestDeliveryHandler_UpdateDelivery_Success(t *testing.T) {
	h := setupDeliveryHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/delivery", h.Handler.UpdateDelivery)

	reqBody := map[string]any{
		"pickup":       true,
		"deliveryType": "own",
		"deliveryKm":   5,
		"deliveryFee":  "2.50",
		"minimumOrder": "15.00",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/delivery",
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
	assert.True(t, response.Pickup)
	assert.Equal(t, restaurant.DeliveryOwn, response.Delivery.Type)
	assert.Equal(t, int16(5), *response.Delivery.RadiusKm)

	var updated restaurant.Restaurant
	err = h.DB.First(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistDelivery])
	assert.Equal(t, restaurant.DeliveryOwn, updated.DeliveryType)
}

func TestDeliveryHandler_UpdateDelivery_Failure_ValidationError(t *testing.T) {
	h := setupDeliveryHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/delivery", h.Handler.UpdateDelivery)

	payload := `{"deliveryType": "invalid"}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/delivery",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestDeliveryHandler_UpdateDelivery_Failure_MissingDeliveryKm(t *testing.T) {
	h := setupDeliveryHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/delivery", h.Handler.UpdateDelivery)

	payload := `{
		"pickup": false,
		"deliveryType": "own",
		"deliveryFee": "2.50",
		"minimumOrder": "10.00"
	}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/delivery",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "DeliveryKm")
}

func TestDeliveryHandler_UpdateDelivery_Failure_Unauthorized(t *testing.T) {
	h := setupDeliveryHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))

	router.PATCH("/restaurants/:id/delivery", h.Handler.UpdateDelivery)

	reqBody := map[string]any{
		"pickup":       true,
		"deliveryType": "own",
		"deliveryKm":   5,
		"deliveryFee":  "2.50",
		"minimumOrder": "15.00",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/delivery",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing X-User-ID header")
}

func TestDeliveryHandler_UpdateDelivery_Failure_Forbidden_WrongRole(t *testing.T) {
	h := setupDeliveryHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "customer"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/delivery", h.Handler.UpdateDelivery)

	reqBody := map[string]any{
		"pickup":       true,
		"deliveryType": "own",
		"deliveryKm":   5,
		"deliveryFee":  "2.50",
		"minimumOrder": "15.00",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/delivery",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "access denied")
}
