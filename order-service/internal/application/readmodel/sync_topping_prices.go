package readmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"order-service/internal/domain/readmodel"
)

// toppingPricesUpdatedPayload mirrors restaurant-service's wire shape — a local, independent copy.
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
	toppingPriceRepo readmodel.ToppingPriceRepository
}

func NewSyncToppingPrices(toppingPriceRepo readmodel.ToppingPriceRepository) *SyncToppingPrices {
	return &SyncToppingPrices{toppingPriceRepo: toppingPriceRepo}
}

func (h *SyncToppingPrices) Handle(event readmodel.EventPayload) error {
	var payload toppingPricesUpdatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	for _, tp := range payload.ToppingPrices {
		extraPrice, err := decimal.NewFromString(tp.ExtraPrice)
		if err != nil {
			return fmt.Errorf("failed to parse topping extra price: %w", err)
		}

		price := readmodel.ToppingPrice{
			RestaurantID: payload.RestaurantID,
			ToppingID:    tp.ToppingID,
			Name:         tp.Name,
			ExtraPrice:   extraPrice,
			UpdatedAt:    payload.UpdatedAt,
		}

		if err := h.toppingPriceRepo.Upsert(context.Background(), price); err != nil {
			return fmt.Errorf("failed to upsert topping price: %w", err)
		}
	}

	return nil
}
