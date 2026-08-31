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

const launchedPizzaSizeID = "b6f5f7de-2b0d-4b6b-9f2b-7c1f7e3f5a01"
const launchedToppingID = "c7a6e8ef-3c1e-4c7c-af3c-8d2f8f4f6b12"

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
		"timezone": "Europe/Berlin",
		"delivery": {"type": "own", "radiusKm": 10, "estimatedMinutesMin": 30, "estimatedMinutesMax": 45, "minimumOrder": "18.00"},
		"currency": "EUR",
		"rating": 4.7,
		"total_reviews": 128,
		"pickup": true,
		"tags": ["vegetarian", "halal"],
		"opening_hours": {
			"monday": [{"open": "11:00", "close": "22:00"}]
		},
		"pizzas": [
			{
				"id": "` + uuid.New().String() + `",
				"name": "Margherita",
				"isVegetarian": true,
				"prices": [
					{"sizeId": "` + launchedPizzaSizeID + `", "diameterCm": 30, "price": "9.99", "isActive": true},
					{"sizeId": "` + uuid.New().String() + `", "diameterCm": 40, "price": "13.99", "isActive": false}
				],
				"toppings": [{"name": "Mozzarella"}, {"name": "Basil"}]
			}
		],
		"topping_prices": [
			{"toppingId": "` + launchedToppingID + `", "name": "Extra Cheese", "extraPrice": "1.50"}
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
	require.NotNil(t, got.DeliveryTimeMin)
	assert.EqualValues(t, 30, *got.DeliveryTimeMin)
	require.NotNil(t, got.DeliveryTimeMax)
	assert.EqualValues(t, 45, *got.DeliveryTimeMax)
	assert.Equal(t, []string{"vegetarian", "halal"}, got.Tags)
	assert.InDelta(t, 4.7, got.Rating, 0.001)
	assert.EqualValues(t, 128, got.TotalReviews)
	assert.True(t, got.UpdatedAt.Equal(time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)))

	assert.InDelta(t, 53.5511, got.Location.Lat, 0.0001)
	assert.InDelta(t, 9.9937, got.Location.Lon, 0.0001)
	assert.Equal(t, "Europe/Berlin", got.Timezone)
	assert.InDelta(t, 18.00, got.MinimumOrder, 0.001)
	assert.Equal(t, []index.IndexedOpeningHours{{Weekday: "monday", Open: "11:00", Close: "22:00"}}, got.OpeningHours)

	require.Len(t, got.Pizzas, 1)
	assert.Equal(t, "Margherita", got.Pizzas[0].Name)
	assert.True(t, got.Pizzas[0].IsVegetarian)
	assert.Equal(t, []string{"Mozzarella", "Basil"}, got.Pizzas[0].Toppings)

	require.Len(t, got.Pizzas[0].Prices, 1, "inactive prices must not be indexed")
	assert.Equal(t, index.IndexedPizzaPrice{
		SizeID:     uuid.MustParse(launchedPizzaSizeID),
		DiameterCm: 30,
		Price:      "9.99",
	}, got.Pizzas[0].Prices[0])

	require.Len(t, got.ToppingPrices, 1, "topping prices set before launch must be seeded into the snapshot")
	assert.Equal(t, index.IndexedToppingPrice{
		ToppingID:  uuid.MustParse(launchedToppingID),
		Name:       "Extra Cheese",
		ExtraPrice: "1.50",
	}, got.ToppingPrices[0])
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
