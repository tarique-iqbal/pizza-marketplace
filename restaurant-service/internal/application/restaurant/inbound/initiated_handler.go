package inbound

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"restaurant-service/internal/domain/restaurant"
)

type RestaurantInitiated struct {
	repo restaurant.RestaurantRepository
}

func NewRestaurantInitiated(repo restaurant.RestaurantRepository) *RestaurantInitiated {
	return &RestaurantInitiated{repo: repo}
}

func (h *RestaurantInitiated) Handle(ctx context.Context, event restaurant.EventPayload) error {
	var payload struct {
		RestaurantID   string `json:"restaurant_id"`
		OwnerID        string `json:"owner_id"`
		RestaurantName string `json:"business_name"`
		VATNumber      string `json:"vat_number"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}

	restaurantID, err := uuid.Parse(payload.RestaurantID)
	if err != nil {
		return err
	}

	ownerID, err := uuid.Parse(payload.OwnerID)
	if err != nil {
		return err
	}

	checklist := restaurant.NewChecklist()
	checklist.Complete(restaurant.ChecklistBasic)

	res := restaurant.NewRestaurant(
		restaurantID,
		ownerID,
		payload.RestaurantName,
		payload.VATNumber,
		checklist,
	)

	return h.repo.Create(ctx, res)
}
