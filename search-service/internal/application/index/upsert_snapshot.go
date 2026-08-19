package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"search-service/internal/domain/index"
)

// restaurantLaunchedPayload mirrors restaurant-service's
// RestaurantLaunchedPayload wire shape. Kept as a local, independent copy —
// search-service never imports restaurant-service's Go code, only agrees
// with it on the JSON contract carried over RabbitMQ.
type restaurantLaunchedPayload struct {
	RestaurantID   uuid.UUID `json:"restaurant_id"`
	RestaurantName string    `json:"restaurant_name"`
	LaunchedAt     time.Time `json:"launched_at"`
	Slug           *string   `json:"slug"`
	Address        struct {
		City string `json:"city"`
	} `json:"address"`
	Lat      *float64 `json:"lat"`
	Lon      *float64 `json:"lon"`
	Delivery struct {
		Type     string `json:"type"`
		RadiusKm *int16 `json:"radiusKm"`
	} `json:"delivery"`
	Currency     string   `json:"currency"`
	Rating       float64  `json:"rating"`
	TotalReviews int32    `json:"total_reviews"`
	Pickup       bool     `json:"pickup"`
	Tags         []string `json:"tags"`
	Pizzas       []struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		IsVegetarian bool      `json:"isVegetarian"`
		Toppings     []struct {
			Name string `json:"name"`
		} `json:"toppings"`
	} `json:"pizzas"`
}

type UpsertSnapshot struct {
	repo index.SearchRepository
}

func NewUpsertSnapshot(repo index.SearchRepository) *UpsertSnapshot {
	return &UpsertSnapshot{repo: repo}
}

func (h *UpsertSnapshot) Handle(event index.EventPayload) error {
	var payload restaurantLaunchedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", event.Name, err)
	}

	if payload.Lat == nil || payload.Lon == nil {
		return fmt.Errorf("%s payload missing lat/lon for restaurant %s", event.Name, payload.RestaurantID)
	}

	if err := h.repo.UpsertSnapshot(context.Background(), toIndexedRestaurant(payload)); err != nil {
		return fmt.Errorf("failed to upsert snapshot: %w", err)
	}

	return nil
}

// toIndexedRestaurant assumes p.Lat/p.Lon are non-nil — Handle validates
// that before calling this.
func toIndexedRestaurant(p restaurantLaunchedPayload) index.IndexedRestaurant {
	location := index.GeoPoint{Lat: *p.Lat, Lon: *p.Lon}

	var slug string
	if p.Slug != nil {
		slug = *p.Slug
	}

	pizzas := make([]index.IndexedPizza, 0, len(p.Pizzas))
	for _, pz := range p.Pizzas {
		toppings := make([]string, 0, len(pz.Toppings))
		for _, t := range pz.Toppings {
			toppings = append(toppings, t.Name)
		}

		pizzas = append(pizzas, index.IndexedPizza{
			ID:           pz.ID,
			Name:         pz.Name,
			IsVegetarian: pz.IsVegetarian,
			Toppings:     toppings,
		})
	}

	return index.IndexedRestaurant{
		ID:           p.RestaurantID,
		Name:         p.RestaurantName,
		Slug:         slug,
		City:         p.Address.City,
		Location:     location,
		Currency:     p.Currency,
		Pickup:       p.Pickup,
		DeliveryType: p.Delivery.Type,
		DeliveryKm:   p.Delivery.RadiusKm,
		Tags:         p.Tags,
		Rating:       p.Rating,
		TotalReviews: p.TotalReviews,
		Pizzas:       pizzas,
	}
}
