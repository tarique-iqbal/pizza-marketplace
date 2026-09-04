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

func newLaunchedPayload(t *testing.T, overrides map[string]any) readmodel.EventPayload {
	body := map[string]any{
		"restaurant_id":   testutil.MustNewID().String(),
		"owner_id":        testutil.MustNewID().String(),
		"restaurant_name": "Pizza Paradise",
		"contact":         map[string]any{"email": "owner@pizzaparadise.de"},
		"lat":             53.5511,
		"lon":             9.9937,
		"delivery": map[string]any{
			"type":         "own",
			"radiusKm":     10,
			"fee":          "2.50",
			"minimumOrder": "10.00",
		},
		"currency": "EUR",
		"pickup":   true,
		"pizzas": []any{
			map[string]any{
				"id":     testutil.MustNewID().String(),
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
				"updatedAt": "2026-09-01T10:00:00Z",
			},
		},
		"topping_prices": []any{
			map[string]any{
				"toppingId":  testutil.MustNewID().String(),
				"name":       "Extra Cheese",
				"extraPrice": "1.50",
			},
		},
		"updated_at": "2026-09-01T10:00:00Z",
	}

	for k, v := range overrides {
		body[k] = v
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return readmodel.EventPayload{Name: "restaurant.launched", Data: data}
}

func TestUpsertRestaurant_Handle(t *testing.T) {
	restaurantRepo := &testutil.MockRestaurantRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	h := appreadmodel.NewUpsertRestaurant(restaurantRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	err := h.Handle(newLaunchedPayload(t, nil))

	require.NoError(t, err)
	require.Len(t, restaurantRepo.Upserted, 1)
	assert.Equal(t, "Pizza Paradise", restaurantRepo.Upserted[0].Name)
	require.Len(t, pizzaRepo.Upserted, 1)
	assert.Equal(t, "Margherita", pizzaRepo.Upserted[0].Name)
	require.Len(t, pizzaPriceRepo.Upserted, 1)
	require.Len(t, toppingPriceRepo.Upserted, 1)
	assert.Equal(t, "Extra Cheese", toppingPriceRepo.Upserted[0].Name)
}

func TestUpsertRestaurant_Handle_SkipsArchivedPizza(t *testing.T) {
	restaurantRepo := &testutil.MockRestaurantRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	h := appreadmodel.NewUpsertRestaurant(restaurantRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	payload := newLaunchedPayload(t, map[string]any{
		"pizzas": []any{
			map[string]any{
				"id":        testutil.MustNewID().String(),
				"name":      "Old Pizza",
				"status":    "archived",
				"prices":    []any{},
				"updatedAt": "2026-09-01T10:00:00Z",
			},
		},
	})

	err := h.Handle(payload)

	require.NoError(t, err)
	assert.Empty(t, pizzaRepo.Upserted)
}

func TestUpsertRestaurant_Handle_SkipsUnpricedPizza(t *testing.T) {
	restaurantRepo := &testutil.MockRestaurantRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	h := appreadmodel.NewUpsertRestaurant(restaurantRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	payload := newLaunchedPayload(t, map[string]any{
		"pizzas": []any{
			map[string]any{
				"id":        testutil.MustNewID().String(),
				"name":      "Unpriced Pizza",
				"status":    "available",
				"prices":    []any{},
				"updatedAt": "2026-09-01T10:00:00Z",
			},
		},
	})

	err := h.Handle(payload)

	require.NoError(t, err)
	assert.Empty(t, pizzaRepo.Upserted)
}

func TestUpsertRestaurant_Handle_MissingContactEmail(t *testing.T) {
	restaurantRepo := &testutil.MockRestaurantRepository{}
	pizzaRepo := &testutil.MockPizzaRepository{}
	pizzaPriceRepo := &testutil.MockPizzaPriceRepository{}
	toppingPriceRepo := &testutil.MockToppingPriceRepository{}

	h := appreadmodel.NewUpsertRestaurant(restaurantRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)

	payload := newLaunchedPayload(t, map[string]any{
		"contact": map[string]any{},
	})

	err := h.Handle(payload)

	assert.Error(t, err)
	assert.Empty(t, restaurantRepo.Upserted)
}
