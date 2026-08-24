package index_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	idxapp "search-service/internal/application/index"
	"search-service/internal/domain/index"
	"search-service/tests/testutil"
)

func launchedPayload(t *testing.T, restaurantID uuid.UUID) []byte {
	t.Helper()

	return []byte(`{
		"restaurant_id": "` + restaurantID.String() + `",
		"restaurant_name": "Anatolische Kueche",
		"updated_at": "2026-08-19T10:00:00Z",
		"slug": "anatolische-kueche",
		"address": {"city": "Hamburg"},
		"lat": 53.5511,
		"lon": 9.9937,
		"delivery": {"type": "own", "radiusKm": 10},
		"currency": "EUR",
		"rating": 4.7,
		"total_reviews": 128,
		"pickup": true,
		"tags": ["vegetarian", "halal"],
		"pizzas": [
			{
				"id": "` + uuid.New().String() + `",
				"name": "Margherita",
				"isVegetarian": true,
				"toppings": [{"name": "Mozzarella"}, {"name": "Basil"}]
			}
		]
	}`)
}

func TestUpsertSnapshot_Success(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewUpsertSnapshot(repo)

	restaurantID := uuid.New()

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.launched",
		Data: launchedPayload(t, restaurantID),
	})
	require.NoError(t, err)

	require.Len(t, repo.Upserted, 1)
	got := repo.Upserted[0]

	assert.Equal(t, restaurantID, got.ID)
	assert.Equal(t, "Anatolische Kueche", got.Name)
	assert.Equal(t, "anatolische-kueche", got.Slug)
	assert.Equal(t, "Hamburg", got.City)
	assert.Equal(t, "EUR", got.Currency)
	assert.True(t, got.Pickup)
	assert.Equal(t, "own", got.DeliveryType)
	require.NotNil(t, got.DeliveryKm)
	assert.EqualValues(t, 10, *got.DeliveryKm)
	assert.Equal(t, []string{"vegetarian", "halal"}, got.Tags)
	assert.InDelta(t, 4.7, got.Rating, 0.001)
	assert.EqualValues(t, 128, got.TotalReviews)
	assert.True(t, got.UpdatedAt.Equal(time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)))

	assert.InDelta(t, 53.5511, got.Location.Lat, 0.0001)
	assert.InDelta(t, 9.9937, got.Location.Lon, 0.0001)

	require.Len(t, got.Pizzas, 1)
	assert.Equal(t, "Margherita", got.Pizzas[0].Name)
	assert.True(t, got.Pizzas[0].IsVegetarian)
	assert.Equal(t, []string{"Mozzarella", "Basil"}, got.Pizzas[0].Toppings)
}

func TestUpsertSnapshot_MissingLatLon_ReturnsError(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewUpsertSnapshot(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.launched",
		Data: []byte(`{"restaurant_id": "` + uuid.New().String() + `", "restaurant_name": "No Coords"}`),
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "missing lat/lon")
	assert.Empty(t, repo.Upserted, "must not silently index a restaurant with no location")
}

func TestUpsertSnapshot_InvalidJSON(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewUpsertSnapshot(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.launched",
		Data: []byte(`not json`),
	})

	require.Error(t, err)
	assert.Empty(t, repo.Upserted)
}

func TestUpsertSnapshot_RepositoryError(t *testing.T) {
	repo := &testutil.MockSearchRepository{UpsertErr: errors.New("es unreachable")}
	handler := idxapp.NewUpsertSnapshot(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.launched",
		Data: launchedPayload(t, uuid.New()),
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "es unreachable")
}
