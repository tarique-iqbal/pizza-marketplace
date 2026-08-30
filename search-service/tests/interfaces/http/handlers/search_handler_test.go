package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"search-service/internal/application/query"
	"search-service/internal/interfaces/http/handlers"
	"search-service/tests/testutil"
)

func setupSearchHandler(
	repo *testutil.MockSearchRepository,
	geocoder *testutil.MockGeocoder,
) *handlers.SearchHandler {
	uc := query.NewSearchRestaurants(repo, geocoder)
	return handlers.NewSearchHandler(uc)
}

func performSearch(h *handlers.SearchHandler, target string) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/search", h.Search)

	req, _ := http.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

const validAddressQS = "street=Main+St&house=1&postalCode=12345&city=Hamburg"

func TestSearchHandler_ResolvesAddressAndSearches(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	geocoder := &testutil.MockGeocoder{Lat: 53.55, Lon: 9.99}
	h := setupSearchHandler(repo, geocoder)

	w := performSearch(h, "/search?"+validAddressQS+"&q=pizza")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1", geocoder.LastAddr.House)
	assert.Equal(t, "Main St", geocoder.LastAddr.Street)
	assert.Equal(t, "Hamburg", geocoder.LastAddr.City)
	assert.Equal(t, "12345", geocoder.LastAddr.PostalCode)
	assert.Equal(t, "pizza", repo.LastQuery.Text)
	assert.InDelta(t, 53.55, repo.LastQuery.Location.Lat, 0.001)
	assert.InDelta(t, 9.99, repo.LastQuery.Location.Lon, 0.001)
}

func TestSearchHandler_MissingAddressField_Returns400(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	geocoder := &testutil.MockGeocoder{}
	h := setupSearchHandler(repo, geocoder)

	w := performSearch(h, "/search?house=1&street=Main+St&city=Hamburg&q=pizza") // no postalCode

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Zero(t, geocoder.CallCount, "must not attempt to geocode an incomplete address")
}

func TestSearchHandler_PassesFulfillmentTagsOpenNowSort_Through(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	geocoder := &testutil.MockGeocoder{Lat: 53.55, Lon: 9.99}
	h := setupSearchHandler(repo, geocoder)

	w := performSearch(
		h,
		"/search?"+validAddressQS+"&fulfillment=pickup&tags=vegan,halal&openNow=true&sort=distance",
	)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pickup", repo.LastQuery.Fulfillment)
	assert.Equal(t, []string{"vegan", "halal"}, repo.LastQuery.Tags)
	assert.True(t, repo.LastQuery.OpenNow)
	assert.Equal(t, "distance", repo.LastQuery.Sort)
}

func TestSearchHandler_NoNewParams_LeavesThemZeroValued(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	geocoder := &testutil.MockGeocoder{Lat: 53.55, Lon: 9.99}
	h := setupSearchHandler(repo, geocoder)

	w := performSearch(h, "/search?"+validAddressQS)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", repo.LastQuery.Fulfillment)
	assert.Empty(t, repo.LastQuery.Tags)
	assert.False(t, repo.LastQuery.OpenNow)
	assert.Equal(t, "", repo.LastQuery.Sort)
}

func TestSearchHandler_EmptyQ_StillSearches(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	geocoder := &testutil.MockGeocoder{Lat: 53.55, Lon: 9.99}
	h := setupSearchHandler(repo, geocoder)

	w := performSearch(h, "/search?"+validAddressQS)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", repo.LastQuery.Text)
}

func TestSearchHandler_GeocodeError_Returns500(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	geocoder := &testutil.MockGeocoder{Err: errors.New("no geocoding results found")}
	h := setupSearchHandler(repo, geocoder)

	w := performSearch(h, "/search?"+validAddressQS+"&q=pizza")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSearchHandler_RepositoryError_Returns500(t *testing.T) {
	repo := &testutil.MockSearchRepository{SearchErr: errors.New("es unreachable")}
	geocoder := &testutil.MockGeocoder{Lat: 53.55, Lon: 9.99}
	h := setupSearchHandler(repo, geocoder)

	w := performSearch(h, "/search?"+validAddressQS+"&q=pizza")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
