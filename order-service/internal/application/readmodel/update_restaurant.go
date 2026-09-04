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

// restaurantUpdatedPayload mirrors restaurant-service's wire shape — a local, independent copy.
type restaurantUpdatedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	OwnerID        uuid.UUID `json:"owner_id"`
	RestaurantName string    `json:"restaurant_name"`
	Contact        struct {
		Email *string `json:"email"`
	} `json:"contact"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Delivery struct {
		Type         string `json:"type"`
		RadiusKm     *int16 `json:"radiusKm"`
		Fee          string `json:"fee"`
		MinimumOrder string `json:"minimumOrder"`
	} `json:"delivery"`
	Currency  string    `json:"currency"`
	Pickup    bool      `json:"pickup"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateRestaurant struct {
	restaurantRepo readmodel.RestaurantRepository
}

func NewUpdateRestaurant(restaurantRepo readmodel.RestaurantRepository) *UpdateRestaurant {
	return &UpdateRestaurant{restaurantRepo: restaurantRepo}
}

func (h *UpdateRestaurant) Handle(event readmodel.EventPayload) error {
	var payload restaurantUpdatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	if payload.Contact.Email == nil {
		return fmt.Errorf("%s payload missing contact email for restaurant %s", event.Name, payload.RestaurantID)
	}

	deliveryFee, err := decimal.NewFromString(payload.Delivery.Fee)
	if err != nil {
		return fmt.Errorf("failed to parse delivery fee: %w", err)
	}

	minimumOrder, err := decimal.NewFromString(payload.Delivery.MinimumOrder)
	if err != nil {
		return fmt.Errorf("failed to parse minimum order: %w", err)
	}

	restaurant := readmodel.Restaurant{
		ID:           payload.RestaurantID,
		OwnerID:      payload.OwnerID,
		Name:         payload.RestaurantName,
		OwnerEmail:   *payload.Contact.Email,
		Lat:          payload.Lat,
		Lon:          payload.Lon,
		DeliveryKm:   payload.Delivery.RadiusKm,
		DeliveryFee:  deliveryFee,
		MinimumOrder: minimumOrder,
		Pickup:       payload.Pickup,
		DeliveryType: readmodel.DeliveryType(payload.Delivery.Type),
		Currency:     payload.Currency,
		UpdatedAt:    payload.UpdatedAt,
	}

	if err := h.restaurantRepo.Upsert(context.Background(), restaurant); err != nil {
		return fmt.Errorf("failed to upsert restaurant: %w", err)
	}

	return nil
}
