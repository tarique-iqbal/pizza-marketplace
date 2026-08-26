package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

// toppingPricesUpdatedPayload mirrors restaurant-service's
// ToppingPricesUpdatedPayload wire shape. Kept as a local, independent copy
// — see upsert_snapshot.go's restaurantLaunchedPayload for why.
type toppingPricesUpdatedPayload struct {
	RestaurantID  uuid.UUID `json:"restaurant_id"`
	ToppingPrices []struct {
		ToppingID  uuid.UUID `json:"toppingId"`
		Name       string    `json:"name"`
		ExtraPrice string    `json:"extraPrice"`
	} `json:"topping_prices"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SyncToppingPrices struct {
	repo index.SearchRepository
}

func NewSyncToppingPrices(repo index.SearchRepository) *SyncToppingPrices {
	return &SyncToppingPrices{repo: repo}
}

func (h *SyncToppingPrices) Handle(event index.EventPayload) error {
	var payload toppingPricesUpdatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	prices := make([]index.IndexedToppingPrice, 0, len(payload.ToppingPrices))
	for _, p := range payload.ToppingPrices {
		prices = append(prices, index.IndexedToppingPrice{
			ToppingID:  p.ToppingID,
			Name:       p.Name,
			ExtraPrice: p.ExtraPrice,
		})
	}

	if err := h.repo.UpdateToppingPrices(
		context.Background(), payload.RestaurantID, prices, payload.UpdatedAt,
	); err != nil {
		return fmt.Errorf("failed to update topping prices: %w", err)
	}

	return nil
}
