package readmodel_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreadmodel "order-service/internal/application/readmodel"
	"order-service/internal/domain/readmodel"
	"order-service/tests/testutil"
)

func newPizzaUpdatedPayload(t *testing.T, pizzaID string, overrides map[string]any) readmodel.EventPayload {
	pizza := map[string]any{
		"id":     pizzaID,
		"name":   "Margherita",
		"status": "available",
		"prices": []any{
			map[string]any{
				"sizeId":     testutil.MustNewID().String(),
				"diameterCm": 26,
				"price":      "7.50",
				"isActive":   true,
			},
		},
	}

	for k, v := range overrides {
		pizza[k] = v
	}

	body := map[string]any{
		"restaurant_id": testutil.MustNewID().String(),
		"pizza":         pizza,
		"updated_at":    "2026-09-01T12:00:00Z",
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return readmodel.EventPayload{Name: "restaurant.pizza_updated", Data: data}
}

func TestSyncPizza_Handle_UpsertsAvailablePizza(t *testing.T) {
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	h := appreadmodel.NewSyncPizza(pizzaRepo, pizzaPriceRepo)

	pizzaID := testutil.MustNewID().String()

	err := h.Handle(newPizzaUpdatedPayload(t, pizzaID, nil))

	require.NoError(t, err)
	require.Len(t, pizzaRepo.Upserted, 1)
	assert.Equal(t, "Margherita", pizzaRepo.Upserted[0].Name)
	require.Len(t, pizzaPriceRepo.Upserted, 1)
	assert.Empty(t, pizzaRepo.Deleted)
}

func TestSyncPizza_Handle_DeletesArchivedPizza(t *testing.T) {
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	h := appreadmodel.NewSyncPizza(pizzaRepo, pizzaPriceRepo)

	pizzaID := testutil.MustNewID().String()

	err := h.Handle(newPizzaUpdatedPayload(t, pizzaID, map[string]any{"status": "archived"}))

	require.NoError(t, err)
	assert.Empty(t, pizzaRepo.Upserted)
	assert.Empty(t, pizzaPriceRepo.Upserted)
	require.Len(t, pizzaRepo.Deleted, 1)
	assert.Equal(t, pizzaID, pizzaRepo.Deleted[0].ID.String())
}

func TestSyncPizza_Handle_DeletesUnpricedPizza(t *testing.T) {
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	h := appreadmodel.NewSyncPizza(pizzaRepo, pizzaPriceRepo)

	pizzaID := testutil.MustNewID().String()

	err := h.Handle(newPizzaUpdatedPayload(t, pizzaID, map[string]any{"prices": []any{}}))

	require.NoError(t, err)
	assert.Empty(t, pizzaRepo.Upserted)
	require.Len(t, pizzaRepo.Deleted, 1)
}
