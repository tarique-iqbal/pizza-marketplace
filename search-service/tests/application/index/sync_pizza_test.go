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

func pizzaUpdatedPayload(
	t *testing.T,
	restaurantID, pizzaID uuid.UUID,
	status string,
	prices string,
) []byte {
	t.Helper()

	return []byte(`{
		"restaurant_id": "` + restaurantID.String() + `",
		"updated_at": "2026-08-25T09:00:00Z",
		"pizza": {
			"id": "` + pizzaID.String() + `",
			"name": "Margherita",
			"isVegetarian": true,
			"status": "` + status + `",
			"prices": ` + prices + `,
			"toppings": [{"name": "Mozzarella"}, {"name": "Basil"}]
		}
	}`)
}

func TestSyncPizza_AvailableAndPriced_Upserts(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewSyncPizza(repo)

	restaurantID := uuid.New()
	pizzaID := uuid.New()
	sizeID := uuid.New()

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.pizza_updated",
		Data: pizzaUpdatedPayload(t, restaurantID, pizzaID, "available", `[
			{"sizeId": "`+sizeID.String()+`", "diameterCm": 30, "price": "9.99", "isActive": true},
			{"sizeId": "`+uuid.New().String()+`", "diameterCm": 40, "price": "13.99", "isActive": false}
		]`),
	})
	require.NoError(t, err)

	require.Empty(t, repo.RemovedPizzas)
	require.Len(t, repo.UpsertedPizzas, 1)
	got := repo.UpsertedPizzas[0]

	assert.Equal(t, restaurantID, got.RestaurantID)
	assert.Equal(t, pizzaID, got.Pizza.ID)
	assert.Equal(t, "Margherita", got.Pizza.Name)
	assert.True(t, got.Pizza.IsVegetarian)
	assert.Equal(t, []string{"Mozzarella", "Basil"}, got.Pizza.Toppings)
	assert.True(t, got.Pizza.UpdatedAt.Equal(time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)))

	require.Len(t, got.Pizza.Prices, 1, "inactive prices must not be indexed")
	assert.Equal(t, index.IndexedPizzaPrice{SizeID: sizeID, DiameterCm: 30, Price: "9.99"}, got.Pizza.Prices[0])
}

func TestSyncPizza_Archived_Removes(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewSyncPizza(repo)

	restaurantID := uuid.New()
	pizzaID := uuid.New()

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.pizza_updated",
		Data: pizzaUpdatedPayload(t, restaurantID, pizzaID, "archived", `[{"isActive": true}]`),
	})
	require.NoError(t, err)

	require.Empty(t, repo.UpsertedPizzas)
	require.Len(t, repo.RemovedPizzas, 1)
	got := repo.RemovedPizzas[0]

	assert.Equal(t, restaurantID, got.RestaurantID)
	assert.Equal(t, pizzaID, got.PizzaID)
}

func TestSyncPizza_NoActivePrice_Removes(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewSyncPizza(repo)

	restaurantID := uuid.New()
	pizzaID := uuid.New()

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.pizza_updated",
		Data: pizzaUpdatedPayload(t, restaurantID, pizzaID, "available", `[{"isActive": false}]`),
	})
	require.NoError(t, err)

	require.Empty(t, repo.UpsertedPizzas)
	require.Len(t, repo.RemovedPizzas, 1)
}

func TestSyncPizza_InvalidJSON(t *testing.T) {
	repo := &testutil.MockSearchRepository{}
	handler := idxapp.NewSyncPizza(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.pizza_updated",
		Data: []byte(`not json`),
	})

	require.Error(t, err)
	assert.Empty(t, repo.UpsertedPizzas)
	assert.Empty(t, repo.RemovedPizzas)
}

func TestSyncPizza_UpsertRepositoryError(t *testing.T) {
	repo := &testutil.MockSearchRepository{UpsertPizzaErr: errors.New("es unreachable")}
	handler := idxapp.NewSyncPizza(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.pizza_updated",
		Data: pizzaUpdatedPayload(t, uuid.New(), uuid.New(), "available", `[{"isActive": true}]`),
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "es unreachable")
}

func TestSyncPizza_RemoveRepositoryError(t *testing.T) {
	repo := &testutil.MockSearchRepository{RemovePizzaErr: errors.New("es unreachable")}
	handler := idxapp.NewSyncPizza(repo)

	err := handler.Handle(index.EventPayload{
		Name: "restaurant.pizza_updated",
		Data: pizzaUpdatedPayload(t, uuid.New(), uuid.New(), "archived", `[]`),
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "es unreachable")
}
