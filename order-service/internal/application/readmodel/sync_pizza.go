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

// pizzaUpdatedPayload mirrors restaurant-service's wire shape — a local, independent copy.
type pizzaUpdatedPayload struct {
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Pizza        struct {
		ID     uuid.UUID `json:"id"`
		Name   string    `json:"name"`
		Status string    `json:"status"`
		Prices []struct {
			SizeID     uuid.UUID `json:"sizeId"`
			DiameterCm int16     `json:"diameterCm"`
			Price      string    `json:"price"`
			IsActive   bool      `json:"isActive"`
		} `json:"prices"`
	} `json:"pizza"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (p pizzaUpdatedPayload) hasActivePrice() bool {
	for _, price := range p.Pizza.Prices {
		if price.IsActive {
			return true
		}
	}

	return false
}

type SyncPizza struct {
	pizzaRepo      readmodel.PizzaRepository
	pizzaPriceRepo readmodel.PizzaPriceRepository
}

func NewSyncPizza(pizzaRepo readmodel.PizzaRepository, pizzaPriceRepo readmodel.PizzaPriceRepository) *SyncPizza {
	return &SyncPizza{pizzaRepo: pizzaRepo, pizzaPriceRepo: pizzaPriceRepo}
}

// Handle deletes an archived or unpriced pizza rather than storing it — it isn't orderable.
func (h *SyncPizza) Handle(event readmodel.EventPayload) error {
	var payload pizzaUpdatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	ctx := context.Background()

	if payload.Pizza.Status == "archived" || !payload.hasActivePrice() {
		if err := h.pizzaRepo.Delete(ctx, payload.Pizza.ID, payload.UpdatedAt); err != nil {
			return fmt.Errorf("failed to delete pizza: %w", err)
		}

		return nil
	}

	pizza := readmodel.Pizza{
		ID:           payload.Pizza.ID,
		RestaurantID: payload.RestaurantID,
		Name:         payload.Pizza.Name,
		Status:       readmodel.PizzaStatus(payload.Pizza.Status),
		UpdatedAt:    payload.UpdatedAt,
	}

	if err := h.pizzaRepo.Upsert(ctx, pizza); err != nil {
		return fmt.Errorf("failed to upsert pizza: %w", err)
	}

	for _, price := range payload.Pizza.Prices {
		amount, err := decimal.NewFromString(price.Price)
		if err != nil {
			return fmt.Errorf("failed to parse pizza price: %w", err)
		}

		pizzaPrice := readmodel.PizzaPrice{
			PizzaID:    payload.Pizza.ID,
			SizeID:     price.SizeID,
			DiameterCm: price.DiameterCm,
			Price:      amount,
			IsActive:   price.IsActive,
			UpdatedAt:  payload.UpdatedAt,
		}

		if err := h.pizzaPriceRepo.Upsert(ctx, pizzaPrice); err != nil {
			return fmt.Errorf("failed to upsert pizza price: %w", err)
		}
	}

	return nil
}
