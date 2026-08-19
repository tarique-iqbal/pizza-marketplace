package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"search-service/internal/application/query"
	"search-service/internal/domain/index"
	"search-service/tests/testutil"
)

func TestSearchRestaurants_ResolvesAddressThenSearches(t *testing.T) {
	repo := &testutil.MockSearchRepository{
		SearchResult: []index.IndexedRestaurant{{Name: "Anatolische Kueche"}},
	}
	geocoder := &testutil.MockGeocoder{Lat: 53.5511, Lon: 9.9937}
	uc := query.NewSearchRestaurants(repo, geocoder)

	addr := index.Address{House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345"}

	results, err := uc.Execute(context.Background(), addr, "pizza")
	require.NoError(t, err)

	assert.Equal(t, addr, geocoder.LastAddr)
	assert.Equal(t, "pizza", repo.LastQuery.Text)
	assert.InDelta(t, 53.5511, repo.LastQuery.Location.Lat, 0.0001)
	assert.InDelta(t, 9.9937, repo.LastQuery.Location.Lon, 0.0001)

	require.Len(t, results, 1)
	assert.Equal(t, "Anatolische Kueche", results[0].Name)
}

func TestSearchRestaurants_GeocodeError_PropagatesAndSkipsSearch(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	geocoder := &testutil.MockGeocoder{Err: errors.New("no geocoding results found")}
	uc := query.NewSearchRestaurants(repo, geocoder)

	_, err := uc.Execute(context.Background(), index.Address{}, "pizza")

	require.Error(t, err)
	assert.ErrorContains(t, err, "no geocoding results found")
	assert.Zero(t, repo.LastQuery, "must not query the repository if the address couldn't be resolved")
}

func TestSearchRestaurants_PropagatesRepositoryError(t *testing.T) {
	repo := &testutil.MockSearchRepository{SearchErr: errors.New("es unreachable")}
	geocoder := &testutil.MockGeocoder{Lat: 53.5511, Lon: 9.9937}
	uc := query.NewSearchRestaurants(repo, geocoder)

	_, err := uc.Execute(context.Background(), index.Address{}, "pizza")

	require.Error(t, err)
	assert.ErrorContains(t, err, "es unreachable")
}
