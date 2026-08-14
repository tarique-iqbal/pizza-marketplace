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

	"restaurant-service/internal/application/payout/commands"
	resapp "restaurant-service/internal/application/restaurant"
	"restaurant-service/internal/domain/payout"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/internal/interfaces/http/handlers"
	"restaurant-service/internal/interfaces/http/middleware"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type payoutHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.PayoutHandler
}

func setupPayoutHandler(t *testing.T) payoutHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	createPayout := commands.NewCreatePayout(restaurantRepo, payoutDetailsRepo, testutil.NoopPublisher{})
	updatePayout := commands.NewUpdatePayout(restaurantRepo, payoutDetailsRepo)
	handler := handlers.NewPayoutHandler(createPayout, updatePayout)

	return payoutHandlerSetup{
		DB:      db.DB,
		Handler: handler,
	}
}

func TestPayoutHandler_CreatePayout_Success(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.POST("/restaurants/:id/payout-details", h.Handler.CreatePayout)

	reqBody := map[string]any{
		"accountHolder": "Mehmet Yilmaz",
		"iban":          "DE89370400440532013000",
		"bic":           "DEUTDEFF",
		"bankName":      "Deutsche Bank",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPost,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusCreated, recorder.Code)

	var response resapp.RestaurantResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, res.ID.String(), response.ID.String())
	assert.Equal(t, "Mehmet Yilmaz", response.Payout.AccountHolder)
	assert.Equal(t, "DE89370400440532013000", response.Payout.IBAN)
	assert.Equal(t, payout.PayoutPending, response.Payout.Status)

	var updated restaurant.Restaurant
	err = h.DB.First(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistPayment])
}

func TestPayoutHandler_CreatePayout_Failure_ValidationError(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.POST("/restaurants/:id/payout-details", h.Handler.CreatePayout)

	payload := `{"accountHolder": "Mehmet Yilmaz", "iban": "not-an-iban", "bic": "DEUTDEFF", "bankName": "Deutsche Bank"}`

	req, _ := http.NewRequest(
		http.MethodPost,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestPayoutHandler_CreatePayout_Failure_Unauthorized(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))

	router.POST("/restaurants/:id/payout-details", h.Handler.CreatePayout)

	reqBody := map[string]any{
		"accountHolder": "Mehmet Yilmaz",
		"iban":          "DE89370400440532013000",
		"bic":           "DEUTDEFF",
		"bankName":      "Deutsche Bank",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPost,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing X-User-ID header")
}

func TestPayoutHandler_CreatePayout_Failure_Forbidden_WrongRole(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "customer"),
		middleware.RequireRole("owner"),
	)

	router.POST("/restaurants/:id/payout-details", h.Handler.CreatePayout)

	reqBody := map[string]any{
		"accountHolder": "Mehmet Yilmaz",
		"iban":          "DE89370400440532013000",
		"bic":           "DEUTDEFF",
		"bankName":      "Deutsche Bank",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPost,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "access denied")
}

func TestPayoutHandler_UpdatePayout_Success(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.POST("/restaurants/:id/payout-details", h.Handler.CreatePayout)
	router.PUT("/restaurants/:id/payout-details", h.Handler.UpdatePayout)

	createPayload := `{"accountHolder": "Mehmet Yilmaz", "iban": "DE89370400440532013000", "bic": "DEUTDEFF", "bankName": "Deutsche Bank"}`

	createReq, _ := http.NewRequest(
		http.MethodPost,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBufferString(createPayload),
	)
	createReq.Header.Set("Content-Type", "application/json")

	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createReq)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	updatePayload := `{"accountHolder": "Ayse Yilmaz", "iban": "GB29NWBK60161331926819", "bic": "NWBKGB2L", "bankName": "NatWest"}`

	updateReq, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBufferString(updatePayload),
	)
	updateReq.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, updateReq)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response resapp.RestaurantResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Ayse Yilmaz", response.Payout.AccountHolder)
	assert.Equal(t, "GB29NWBK60161331926819", response.Payout.IBAN)
	assert.Equal(t, payout.PayoutPending, response.Payout.Status)
}

func TestPayoutHandler_UpdatePayout_Failure_ValidationError(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PUT("/restaurants/:id/payout-details", h.Handler.UpdatePayout)

	payload := `{"accountHolder": "Mehmet Yilmaz", "iban": "not-an-iban", "bic": "DEUTDEFF", "bankName": "Deutsche Bank"}`

	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestPayoutHandler_UpdatePayout_Failure_NothingPending(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PUT("/restaurants/:id/payout-details", h.Handler.UpdatePayout)

	payload := `{"accountHolder": "Mehmet Yilmaz", "iban": "DE89370400440532013000", "bic": "DEUTDEFF", "bankName": "Deutsche Bank"}`

	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBufferString(payload),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestPayoutHandler_UpdatePayout_Failure_Forbidden_WrongRole(t *testing.T) {
	h := setupPayoutHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "customer"),
		middleware.RequireRole("owner"),
	)

	router.PUT("/restaurants/:id/payout-details", h.Handler.UpdatePayout)

	payload := `{"accountHolder": "Mehmet Yilmaz", "iban": "DE89370400440532013000", "bic": "DEUTDEFF", "bankName": "Deutsche Bank"}`

	req, _ := http.NewRequest(
		http.MethodPut,
		"/restaurants/"+res.ID.String()+"/payout-details",
		bytes.NewBufferString(payload),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "access denied")
}
