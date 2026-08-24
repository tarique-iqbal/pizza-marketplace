package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

type restaurantUpdatedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	UpdatedAt      time.Time `json:"updated_at"`
	Slug           string    `json:"slug"`
	Address        struct {
		City string `json:"city"`
	} `json:"address"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Delivery struct {
		Type     string `json:"type"`
		RadiusKm *int16 `json:"radiusKm"`
	} `json:"delivery"`
	Currency     string   `json:"currency"`
	Rating       float64  `json:"rating"`
	TotalReviews int32    `json:"total_reviews"`
	Pickup       bool     `json:"pickup"`
	Tags         []string `json:"tags"`
}

type UpdateRestaurantFields struct {
	repo index.SearchRepository
}

func NewUpdateRestaurantFields(repo index.SearchRepository) *UpdateRestaurantFields {
	return &UpdateRestaurantFields{repo: repo}
}

func (h *UpdateRestaurantFields) Handle(event index.EventPayload) error {
	var payload restaurantUpdatedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	if err := h.repo.UpdateFields(context.Background(), payload.RestaurantID, toRestaurantFields(payload)); err != nil {
		return fmt.Errorf("failed to update restaurant fields: %w", err)
	}

	return nil
}

func toRestaurantFields(p restaurantUpdatedPayload) index.RestaurantFields {
	return index.RestaurantFields{
		Name:         p.RestaurantName,
		Slug:         p.Slug,
		City:         p.Address.City,
		Location:     index.GeoPoint{Lat: p.Lat, Lon: p.Lon},
		Currency:     p.Currency,
		Pickup:       p.Pickup,
		DeliveryType: p.Delivery.Type,
		DeliveryKm:   p.Delivery.RadiusKm,
		Tags:         p.Tags,
		Rating:       p.Rating,
		TotalReviews: p.TotalReviews,
		UpdatedAt:    p.UpdatedAt,
	}
}
