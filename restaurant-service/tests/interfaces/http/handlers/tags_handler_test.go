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

type tagsHandlerSetup struct {
	DB      *gorm.DB
	Handler *handlers.TagsHandler
}

func setupTagsHandler(t *testing.T) tagsHandlerSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))

	restaurantRepo := persistence.NewRestaurantRepository(db.DB)
	payoutDetailsRepo := persistence.NewPayoutDetailsRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	handler := handlers.NewTagsHandler(
		commands.NewUpdateTags(db.DB, restaurantRepo, payoutDetailsRepo, outboxRepo),
	)

	return tagsHandlerSetup{DB: db.DB, Handler: handler}
}

func tagsRouter(h *handlers.TagsHandler, ownerID, role string) *gin.Engine {
	router := gin.Default()
	router.Use(MockAuthMiddleware(ownerID, role), middleware.RequireRole("owner"))

	router.PATCH("/restaurants/:id/tags", h.UpdateTags)

	return router
}

func TestTagsHandler_UpdateTags_Success(t *testing.T) {
	h := setupTagsHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := tagsRouter(h.Handler, res.OwnerID.String(), "owner")

	body, _ := json.Marshal(map[string]any{"tags": []string{"vegan", "halal"}})
	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/tags",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response resapp.RestaurantResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, []string{"vegan", "halal"}, response.Tags)

	var updated restaurant.Restaurant
	require.NoError(t, h.DB.Take(&updated, "id = ?", res.ID).Error)
	assert.Equal(t, []restaurant.RestaurantTag{restaurant.TagVegan, restaurant.TagHalal}, updated.Tags)
}

func TestTagsHandler_UpdateTags_ValidationError_UnknownTag(t *testing.T) {
	h := setupTagsHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := tagsRouter(h.Handler, res.OwnerID.String(), "owner")

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/tags",
		bytes.NewBufferString(`{"tags": ["kosher"]}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestTagsHandler_UpdateTags_ValidationError_DuplicateTag(t *testing.T) {
	h := setupTagsHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := tagsRouter(h.Handler, res.OwnerID.String(), "owner")

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/tags",
		bytes.NewBufferString(`{"tags": ["vegan", "vegan"]}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "errors")
}

func TestTagsHandler_Unauthorized(t *testing.T) {
	h := setupTagsHandler(t)

	var res restaurant.Restaurant
	require.NoError(t, h.DB.Where("slug = ?", "anatolische-kueche").Take(&res).Error)

	router := gin.Default()
	router.Use(middleware.AuthMiddleware())
	router.Use(middleware.RequireRole("owner"))
	router.PATCH("/restaurants/:id/tags", h.Handler.UpdateTags)

	req, _ := http.NewRequest(
		http.MethodPatch,
		"/restaurants/"+res.ID.String()+"/tags",
		bytes.NewBufferString(`{"tags": []}`),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
