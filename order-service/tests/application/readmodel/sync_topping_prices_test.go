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

func newToppingPricesUpdatedPayload(t *testing.T, overrides map[string]any) readmodel.EventPayload {
	body := map[string]any{
		"restaurant_id": testutil.MustNewID().String(),
		"topping_prices": []any{
			map[string]any{
				"toppingId":  testutil.MustNewID().String(),
				"name":       "Extra Cheese",
				"extraPrice": "1.50",
			},
			map[string]any{
				"toppingId":  testutil.MustNewID().String(),
				"name":       "Mushrooms",
				"extraPrice": "1.00",
			},
		},
		"updated_at": "2026-09-01T12:00:00Z",
	}

	for k, v := range overrides {
		body[k] = v
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return readmodel.EventPayload{Name: "restaurant.topping_prices_updated", Data: data}
}

func TestSyncToppingPrices_Handle(t *testing.T) {
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}
	h := appreadmodel.NewSyncToppingPrices(toppingPriceRepo)

	err := h.Handle(newToppingPricesUpdatedPayload(t, nil))

	require.NoError(t, err)
	require.Len(t, toppingPriceRepo.Upserted, 2)
	assert.Equal(t, "Extra Cheese", toppingPriceRepo.Upserted[0].Name)
	assert.Equal(t, "Mushrooms", toppingPriceRepo.Upserted[1].Name)
}

func TestSyncToppingPrices_Handle_Empty(t *testing.T) {
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}
	h := appreadmodel.NewSyncToppingPrices(toppingPriceRepo)

	err := h.Handle(newToppingPricesUpdatedPayload(t, map[string]any{"topping_prices": []any{}}))

	require.NoError(t, err)
	assert.Empty(t, toppingPriceRepo.Upserted)
}
