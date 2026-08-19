package elasticsearch_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"search-service/internal/domain/index"
	esinfra "search-service/internal/infrastructure/elasticsearch"
	"search-service/tests/testutil"
)

func int16Ptr(v int16) *int16 {
	return &v
}

func upsert(t *testing.T, repo *esinfra.SearchRepository, r index.IndexedRestaurant) {
	t.Helper()
	require.NoError(t, repo.UpsertSnapshot(context.Background(), r))
}

func TestSearchRepository_UpsertAndSearch_TextMatch(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	upsert(t, repo, index.IndexedRestaurant{
		ID:         uuid.New(),
		Name:       "Pizzeria Napoli",
		Slug:       "napoli",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Napoli",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Pizzeria Napoli", results[0].Name)
}

func TestSearchRepository_Search_FuzzyMatch(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	upsert(t, repo, index.IndexedRestaurant{
		ID:         uuid.New(),
		Name:       "Pizzeria Roma",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Pizzeriaa", // typo
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Pizzeria Roma", results[0].Name)
}

func TestSearchRepository_Search_EmptyText_MatchesAllInRange(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	upsert(t, repo, index.IndexedRestaurant{
		ID:         uuid.New(),
		Name:       "Anything At All",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestSearchRepository_Search_OutOfDeliveryRange_Excluded(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	upsert(t, repo, index.IndexedRestaurant{
		ID:         uuid.New(),
		Name:       "Pizzeria Napoli",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937}, // Hamburg
		DeliveryKm: int16Ptr(10),
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Napoli",
		Location: index.GeoPoint{Lat: 52.5200, Lon: 13.4050}, // Berlin, ~250km away
	})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchRepository_Search_NoDeliveryKm_Excluded(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	upsert(t, repo, index.IndexedRestaurant{
		ID:         uuid.New(),
		Name:       "Pickup Only Pizzeria",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: nil,
		Pickup:     true,
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Pickup",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937}, // same coordinates
	})
	require.NoError(t, err)
	assert.Empty(t, results, "a restaurant with no configured delivery radius must never match, even at distance 0")
}

func TestSearchRepository_Search_RatingBoostsOrdering(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	upsert(t, repo, index.IndexedRestaurant{
		ID:         uuid.New(),
		Name:       "Pizzeria Low Rated",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		Rating:     3.0,
	})
	upsert(t, repo, index.IndexedRestaurant{
		ID:         uuid.New(),
		Name:       "Pizzeria High Rated",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		Rating:     4.9,
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Pizzeria",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "Pizzeria High Rated", results[0].Name)
	assert.Equal(t, "Pizzeria Low Rated", results[1].Name)
}

func TestSearchRepository_UpsertSnapshot_Overwrites(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Old Name",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
	})
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "New Name",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Name",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1, "upserting the same restaurant ID must replace, not duplicate, the document")
	assert.Equal(t, "New Name", results[0].Name)
	assert.Equal(t, id, results[0].ID)
}
