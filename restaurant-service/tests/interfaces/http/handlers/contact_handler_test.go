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

type contactHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.ContactHandler
}

func setupContactHandler(t *testing.T) contactHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	_ = fixtures.LoadRestaurantFixtures(t, db.DB)

	repo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	updateContact := commands.NewUpdateContact(db.DB, repo, payoutDetailsRepo, outboxRepo)
	handler := handlers.NewContactHandler(updateContact)

	return contactHandlerSetup{
		DB:      db.DB,
		Handler: handler,
	}
}

func TestContactHandler_UpdateContact_Success(t *testing.T) {
	h := setupContactHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/contact", h.Handler.UpdateContact)

	reqBody := map[string]any{
		"email":   "owner@example.com",
		"phone":   "+49 40 12345678",
		"website": "https://example.com",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/contact",
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
	assert.Equal(t, "owner@example.com", *response.Contact.Email)
	assert.Equal(t, "+49 40 12345678", *response.Contact.Phone)
	assert.Equal(t, "https://example.com", *response.Contact.Website)

	var updated restaurant.Restaurant
	err = h.DB.First(&updated, "id = ?", res.ID).Error
	require.NoError(t, err)

	assert.True(t, updated.Checklist[restaurant.ChecklistContact])
	assert.Equal(t, "owner@example.com", *updated.Email)
}

func TestContactHandler_UpdateContact_Failure_ValidationError(t *testing.T) {
	h := setupContactHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/contact", h.Handler.UpdateContact)

	payload := `{"email": "not-an-email"}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/contact",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestContactHandler_UpdateContact_Failure_ValidationError_MissingRequiredFields(t *testing.T) {
	h := setupContactHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/contact", h.Handler.UpdateContact)

	payload := `{"website": "https://example.com"}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/contact",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestContactHandler_UpdateContact_Failure_ValidationError_InvalidPhone(t *testing.T) {
	h := setupContactHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "owner"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/contact", h.Handler.UpdateContact)

	payload := `{"phone": "asdf"}`

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/contact",
		bytes.NewBufferString(payload),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestContactHandler_UpdateContact_Failure_Unauthorized(t *testing.T) {
	h := setupContactHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))

	router.PATCH("/restaurants/:id/contact", h.Handler.UpdateContact)

	reqBody := map[string]any{
		"email":   "owner@example.com",
		"phone":   "+49 40 12345678",
		"website": "https://example.com",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/contact",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "missing X-User-ID header")
}

func TestContactHandler_UpdateContact_Failure_Forbidden_WrongRole(t *testing.T) {
	h := setupContactHandler(t)

	var res restaurant.Restaurant
	err := h.DB.First(&res).Error
	require.NoError(t, err)

	router := gin.Default()

	router.Use(
		MockAuthMiddleware(res.OwnerID.String(), "customer"),
		middleware.RequireRole("owner"),
	)

	router.PATCH("/restaurants/:id/contact", h.Handler.UpdateContact)

	reqBody := map[string]any{
		"email":   "owner@example.com",
		"phone":   "+49 40 12345678",
		"website": "https://example.com",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/contact",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "access denied")
}
