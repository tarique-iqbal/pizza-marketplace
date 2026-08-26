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

func toppingPricesUpdatedPayload(t *testing.T, restaurantID, toppingID uuid.UUID) []byte {
	t.Helper()

	return []byte(`{
		"restaurant_id": "` + restaurantID.String() + `",
		"updated_at": "2026-08-26T09:00:00Z",
		"topping_prices": [
			{"toppingId": "` + toppingID.String() + `", "name": "Extra Cheese", "extraPrice": "1.50"}
		]
	}`)
}

func TestSyncToppingPrices_Success(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewSyncToppingPrices(repo)

	restaurantID := uuid.New()
	toppingID := uuid.New()

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.topping_prices_updated",
		Data: toppingPricesUpdatedPayload(t, restaurantID, toppingID),
	})
	require.NoError(t, err)

	require.Len(t, repo.UpdatedToppingPrices, 1)
	got := repo.UpdatedToppingPrices[0]

	assert.Equal(t, restaurantID, got.RestaurantID)
	require.Len(t, got.Prices, 1)
	assert.Equal(t, index.IndexedToppingPrice{ToppingID: toppingID, Name: "Extra Cheese", ExtraPrice: "1.50"}, got.Prices[0])
	assert.True(t, got.UpdatedAt.Equal(time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)))
}

func TestSyncToppingPrices_EmptyList(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewSyncToppingPrices(repo)

	restaurantID := uuid.New()

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.topping_prices_updated",
		Data: []byte(`{"restaurant_id": "` + restaurantID.String() + `", "updated_at": "2026-08-26T09:00:00Z", "topping_prices": []}`),
	})
	require.NoError(t, err)

	require.Len(t, repo.UpdatedToppingPrices, 1)
	assert.Empty(t, repo.UpdatedToppingPrices[0].Prices)
}

func TestSyncToppingPrices_InvalidJSON(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewSyncToppingPrices(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.topping_prices_updated",
		Data: []byte(`not json`),
	})

	require.Error(t, err)
	assert.Empty(t, repo.UpdatedToppingPrices)
}

func TestSyncToppingPrices_RepositoryError(t *testing.T) {
	repo := &testutil.MockSearchRepository{UpdateToppingPricesErr: errors.New("es unreachable")}
	handler := idxapp.NewSyncToppingPrices(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.topping_prices_updated",
		Data: toppingPricesUpdatedPayload(t, uuid.New(), uuid.New()),
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "es unreachable")
}
