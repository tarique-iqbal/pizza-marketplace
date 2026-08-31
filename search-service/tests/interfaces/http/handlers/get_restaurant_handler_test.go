package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"search-service/internal/application/query"
	"search-service/internal/domain/index"
	"search-service/internal/interfaces/http/handlers"
	apperr "search-service/internal/shared/errors"
	"search-service/tests/testutil"
)

func setupGetRestaurantHandler(repo *testutil.MockSearchRepository) *handlers.GetRestaurantHandler {
	uc := query.NewGetRestaurant(repo)
	return handlers.NewGetRestaurantHandler(uc)
}

func performGetRestaurant(h *handlers.GetRestaurantHandler, target string) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/search/restaurant/:id", h.Get)

	req, _ := http.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestGetRestaurantHandler_Found_Returns200(t *testing.T) {
	id := uuid.New()
	repo := &testutil.MockSearchRepository{
		FindByIDResult: index.IndexedRestaurant{ID: id, Name: "Pizzeria Bella"},
	}
	h := setupGetRestaurantHandler(repo)

	w := performGetRestaurant(h, "/search/restaurant/"+id.String())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Pizzeria Bella")
}

func TestGetRestaurantHandler_InvalidID_Returns400(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	h := setupGetRestaurantHandler(repo)

	w := performGetRestaurant(h, "/search/restaurant/not-a-uuid")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRestaurantHandler_NotFound_Returns404(t *testing.T) {
	id := uuid.New()
	repo := &testutil.MockSearchRepository{
		FindByIDErr: fmt.Errorf("restaurant %s: %w", id, apperr.ErrNotFound),
	}
	h := setupGetRestaurantHandler(repo)

	w := performGetRestaurant(h, "/search/restaurant/"+id.String())

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetRestaurantHandler_RepositoryError_Returns500(t *testing.T) {
	id := uuid.New()
	repo := &testutil.MockSearchRepository{
		FindByIDErr: fmt.Errorf("es unreachable"),
	}
	h := setupGetRestaurantHandler(repo)

	w := performGetRestaurant(h, "/search/restaurant/"+id.String())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
