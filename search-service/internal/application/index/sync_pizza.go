package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

// pizzaUpdatedPayload mirrors restaurant-service's PizzaUpdatedPayload wire
// shape. Kept as a local, independent copy — see upsert_snapshot.go's
// restaurantLaunchedPayload for why.
type pizzaUpdatedPayload struct {
	RestaurantID uuid.UUID `json:"restaurant_id"`
	UpdatedAt    time.Time `json:"updated_at"`
	Pizza        struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		IsVegetarian bool      `json:"isVegetarian"`
		Status       string    `json:"status"`
		Prices       []struct {
			IsActive bool `json:"isActive"`
		} `json:"prices"`
		Toppings []struct {
			Name string `json:"name"`
		} `json:"toppings"`
	} `json:"pizza"`
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
	repo index.SearchRepository
}

func NewSyncPizza(repo index.SearchRepository) *SyncPizza {
	return &SyncPizza{repo: repo}
}

// Handle mirrors restaurant.pizza_updated into the indexed restaurant's
// pizzas array. An archived or unpriced pizza isn't orderable, so it's
// removed from the index rather than upserted — matching restaurant-
// service's own public-facing filter for unsellable pizzas.
func (h *SyncPizza) Handle(event index.EventPayload) error {
	var payload pizzaUpdatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	if payload.Pizza.Status == "archived" || !payload.hasActivePrice() {
		if err := h.repo.RemovePizza(
			context.Background(), payload.RestaurantID, payload.Pizza.ID, payload.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to remove pizza: %w", err)
		}

		return nil
	}

	if err := h.repo.UpsertPizza(context.Background(), payload.RestaurantID, toIndexedPizza(payload)); err != nil {
		return fmt.Errorf("failed to upsert pizza: %w", err)
	}

	return nil
}

func toIndexedPizza(p pizzaUpdatedPayload) index.IndexedPizza {
	toppings := make([]string, 0, len(p.Pizza.Toppings))
	for _, t := range p.Pizza.Toppings {
		toppings = append(toppings, t.Name)
	}

	return index.IndexedPizza{
		ID:           p.Pizza.ID,
		Name:         p.Pizza.Name,
		IsVegetarian: p.Pizza.IsVegetarian,
		Toppings:     toppings,
		UpdatedAt:    p.UpdatedAt,
	}
}
