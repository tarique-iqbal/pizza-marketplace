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

func newUpdatedPayload(t *testing.T, overrides map[string]any) readmodel.EventPayload {
	body := map[string]any{
		"restaurant_id":   testutil.MustNewID().String(),
		"owner_id":        testutil.MustNewID().String(),
		"restaurant_name": "Pizza Paradise Updated",
		"contact":         map[string]any{"email": "owner@pizzaparadise.de"},
		"lat":             53.5511,
		"lon":             9.9937,
		"delivery": map[string]any{
			"type":         "own",
			"radiusKm":     10,
			"fee":          "2.50",
			"minimumOrder": "10.00",
		},
		"currency":   "EUR",
		"pickup":     true,
		"updated_at": "2026-09-01T12:00:00Z",
	}

	for k, v := range overrides {
		body[k] = v
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return readmodel.EventPayload{Name: "restaurant.updated", Data: data}
}

func TestUpdateRestaurant_Handle(t *testing.T) {
	restaurantRepo := &testutil.MockRestaurantRepository{}
	h := appreadmodel.NewUpdateRestaurant(restaurantRepo)

	err := h.Handle(newUpdatedPayload(t, nil))

	require.NoError(t, err)
	require.Len(t, restaurantRepo.Upserted, 1)
	assert.Equal(t, "Pizza Paradise Updated", restaurantRepo.Upserted[0].Name)
}

func TestUpdateRestaurant_Handle_MissingContactEmail(t *testing.T) {
	restaurantRepo := &testutil.MockRestaurantRepository{}
	h := appreadmodel.NewUpdateRestaurant(restaurantRepo)

	err := h.Handle(newUpdatedPayload(t, map[string]any{"contact": map[string]any{}}))

	assert.Error(t, err)
	assert.Empty(t, restaurantRepo.Upserted)
}
