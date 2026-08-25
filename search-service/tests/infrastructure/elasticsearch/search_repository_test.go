package elasticsearch_test

import (
	"context"
	"testing"
	"time"

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
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
	})
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "New Name",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 1, 0, time.UTC),
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

func TestSearchRepository_UpsertSnapshot_StaleRedelivery_Ignored(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Fresh Name",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 5, 0, time.UTC),
	})

	// A stale, retried redelivery of an OLDER event must not clobber the
	// newer write above — this is the whole point of the updatedAt guard.
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Stale Name",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
	})
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Name",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Fresh Name", results[0].Name, "a stale redelivery must not overwrite newer indexed data")
}

func TestSearchRepository_UpdateFields_UpdatesFieldsAndPreservesPizzas(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	pizzaID := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
		Pizzas: []index.IndexedPizza{
			{ID: pizzaID, Name: "Margherita", IsVegetarian: true},
		},
	})

	require.NoError(t, repo.UpdateFields(context.Background(), id, index.RestaurantFields{
		Name:       "Pizzeria Renamed",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 5, 0, time.UTC),
	}))
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Renamed",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Pizzeria Renamed", results[0].Name)
	require.Len(t, results[0].Pizzas, 1, "a restaurant-field update must never touch the indexed menu")
	assert.Equal(t, "Margherita", results[0].Pizzas[0].Name)
}

func TestSearchRepository_UpdateFields_StaleRedelivery_Ignored(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
	})

	require.NoError(t, repo.UpdateFields(context.Background(), id, index.RestaurantFields{
		Name:       "Pizzeria Fresh",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 10, 0, time.UTC),
	}))

	// A stale, retried redelivery of an OLDER restaurant.updated event must
	// be dropped, not overwrite the fresher name set above.
	require.NoError(t, repo.UpdateFields(context.Background(), id, index.RestaurantFields{
		Name:       "Pizzeria Stale",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 5, 0, time.UTC),
	}))
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Pizzeria",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Pizzeria Fresh", results[0].Name, "a stale redelivery must not overwrite newer indexed data")
}

func TestSearchRepository_UpdateFields_DocumentMissing_ReturnsError(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	err := repo.UpdateFields(context.Background(), uuid.New(), index.RestaurantFields{
		Name:      "Ghost",
		UpdatedAt: time.Now(),
	})

	require.Error(t, err, "an update for a restaurant not yet indexed by launch must not silently create one")
}

func TestSearchRepository_UpsertPizza_AddsNewPizza(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	pizzaID := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
	})

	require.NoError(t, repo.UpsertPizza(context.Background(), id, index.IndexedPizza{
		ID:           pizzaID,
		Name:         "Margherita",
		IsVegetarian: true,
		Toppings:     []string{"Mozzarella"},
		UpdatedAt:    time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC),
	}))
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Original",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Pizzas, 1)
	assert.Equal(t, "Margherita", results[0].Pizzas[0].Name)
}

func TestSearchRepository_UpsertPizza_ReplacesExistingByID(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	pizzaID := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
		Pizzas: []index.IndexedPizza{
			{ID: pizzaID, Name: "Margherita", UpdatedAt: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)},
		},
	})

	require.NoError(t, repo.UpsertPizza(context.Background(), id, index.IndexedPizza{
		ID:        pizzaID,
		Name:      "Margherita Deluxe",
		UpdatedAt: time.Date(2026, 8, 25, 9, 0, 5, 0, time.UTC),
	}))
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Original",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Pizzas, 1, "replacing an existing pizza by id must not duplicate it")
	assert.Equal(t, "Margherita Deluxe", results[0].Pizzas[0].Name)
}

func TestSearchRepository_UpsertPizza_StaleRedelivery_Ignored(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	pizzaID := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
		Pizzas: []index.IndexedPizza{
			{ID: pizzaID, Name: "Fresh Price", UpdatedAt: time.Date(2026, 8, 25, 9, 0, 10, 0, time.UTC)},
		},
	})

	// A stale, retried redelivery of an OLDER restaurant.pizza_updated event
	// must not clobber the fresher pizza state set above.
	require.NoError(t, repo.UpsertPizza(context.Background(), id, index.IndexedPizza{
		ID:        pizzaID,
		Name:      "Stale Price",
		UpdatedAt: time.Date(2026, 8, 25, 9, 0, 5, 0, time.UTC),
	}))
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Original",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Pizzas, 1)
	assert.Equal(t, "Fresh Price", results[0].Pizzas[0].Name, "a stale redelivery must not overwrite newer pizza data")
}

func TestSearchRepository_UpsertPizza_DocumentMissing_ReturnsError(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	err := repo.UpsertPizza(context.Background(), uuid.New(), index.IndexedPizza{
		ID:        uuid.New(),
		Name:      "Ghost Pizza",
		UpdatedAt: time.Now(),
	})

	require.Error(t, err, "a pizza update for a restaurant not yet indexed by launch must not silently create one")
}

func TestSearchRepository_RemovePizza_RemovesExisting(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	pizzaID := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
		Pizzas: []index.IndexedPizza{
			{ID: pizzaID, Name: "Margherita", UpdatedAt: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)},
		},
	})

	require.NoError(
		t,
		repo.RemovePizza(context.Background(), id, pizzaID, time.Date(2026, 8, 25, 9, 0, 5, 0, time.UTC)),
	)
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Original",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Empty(t, results[0].Pizzas, "an archived/unpriced pizza must be removed from the indexed menu")
}

func TestSearchRepository_RemovePizza_MissingPizza_Noop(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
	})

	err := repo.RemovePizza(context.Background(), id, uuid.New(), time.Now())
	require.NoError(t, err, "removing a pizza that was never indexed must be idempotent, not an error")
}

func TestSearchRepository_RemovePizza_StaleRedelivery_Ignored(t *testing.T) {
	es := testutil.ES(t)
	repo := esinfra.NewSearchRepository(es)

	id := uuid.New()
	pizzaID := uuid.New()
	upsert(t, repo, index.IndexedRestaurant{
		ID:         id,
		Name:       "Pizzeria Original",
		Location:   index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
		DeliveryKm: int16Ptr(10),
		UpdatedAt:  time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC),
		Pizzas: []index.IndexedPizza{
			{ID: pizzaID, Name: "Margherita", UpdatedAt: time.Date(2026, 8, 25, 9, 0, 10, 0, time.UTC)},
		},
	})

	// A stale, retried redelivery of an OLDER removal must not drop a pizza
	// that was re-added (a fresher upsert) after the removal was published.
	require.NoError(
		t,
		repo.RemovePizza(context.Background(), id, pizzaID, time.Date(2026, 8, 25, 9, 0, 5, 0, time.UTC)),
	)
	testutil.RefreshIndex(t, es, esinfra.IndexName)

	results, err := repo.Search(context.Background(), index.SearchQuery{
		Text:     "Original",
		Location: index.GeoPoint{Lat: 53.5511, Lon: 9.9937},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, results[0].Pizzas, 1, "a stale removal must not drop a fresher pizza entry")
}
