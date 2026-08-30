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

func updatedPayload(t *testing.T, restaurantID uuid.UUID) []byte {
	t.Helper()

	return []byte(`{
		"restaurant_id": "` + restaurantID.String() + `",
		"restaurant_name": "Anatolische Kueche",
		"updated_at": "2026-08-24T09:00:00Z",
		"slug": "anatolische-kueche",
		"address": {"city": "Berlin"},
		"lat": 52.5200,
		"lon": 13.4050,
		"timezone": "Europe/Berlin",
		"delivery": {"type": "own", "radiusKm": 15, "minimumOrder": "20.00"},
		"currency": "EUR",
		"rating": 4.8,
		"total_reviews": 200,
		"pickup": false,
		"tags": ["vegan"],
		"opening_hours": {
			"tuesday": [{"open": "10:00", "close": "20:00"}]
		}
	}`)
}

func TestUpdateRestaurantFields_Success(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewUpdateRestaurantFields(repo)

	restaurantID := uuid.New()

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.updated",
		Data: updatedPayload(t, restaurantID),
	})
	require.NoError(t, err)

	require.Len(t, repo.UpdatedFields, 1)
	got := repo.UpdatedFields[0]

	assert.Equal(t, restaurantID, got.ID)
	assert.Equal(t, "Anatolische Kueche", got.Fields.Name)
	assert.Equal(t, "anatolische-kueche", got.Fields.Slug)
	assert.Equal(t, "Berlin", got.Fields.City)
	assert.Equal(t, "EUR", got.Fields.Currency)
	assert.False(t, got.Fields.Pickup)
	assert.Equal(t, "own", got.Fields.DeliveryType)
	require.NotNil(t, got.Fields.DeliveryKm)
	assert.EqualValues(t, 15, *got.Fields.DeliveryKm)
	assert.Equal(t, []string{"vegan"}, got.Fields.Tags)
	assert.InDelta(t, 4.8, got.Fields.Rating, 0.001)
	assert.EqualValues(t, 200, got.Fields.TotalReviews)
	assert.InDelta(t, 52.5200, got.Fields.Location.Lat, 0.0001)
	assert.InDelta(t, 13.4050, got.Fields.Location.Lon, 0.0001)
	assert.Equal(t, "Europe/Berlin", got.Fields.Timezone)
	assert.InDelta(t, 20.00, got.Fields.MinimumOrder, 0.001)
	assert.Equal(t, []index.IndexedOpeningHours{{Weekday: "tuesday", Open: "10:00", Close: "20:00"}}, got.Fields.OpeningHours)
	assert.True(t, got.Fields.UpdatedAt.Equal(time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)))
}

func TestUpdateRestaurantFields_InvalidJSON(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewUpdateRestaurantFields(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.updated",
		Data: []byte(`not json`),
	})

	require.Error(t, err)
	assert.Empty(t, repo.UpdatedFields)
}

func TestUpdateRestaurantFields_RepositoryError(t *testing.T) {
	repo := &testutil.MockSearchRepository{UpdateErr: errors.New("es unreachable")}
	handler := idxapp.NewUpdateRestaurantFields(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.updated",
		Data: updatedPayload(t, uuid.New()),
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "es unreachable")
}
