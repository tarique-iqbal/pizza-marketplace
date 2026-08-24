package index

import (
	"time"

	"github.com/google/uuid"
)

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type IndexedPizza struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	IsVegetarian bool      `json:"isVegetarian"`
	Toppings     []string  `json:"toppings"`
}

type IndexedRestaurant struct {
	ID           uuid.UUID      `json:"id"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	City         string         `json:"city"`
	Location     GeoPoint       `json:"location"`
	Currency     string         `json:"currency"`
	Pickup       bool           `json:"pickup"`
	DeliveryType string         `json:"deliveryType"`
	DeliveryKm   *int16         `json:"deliveryKm,omitempty"`
	Tags         []string       `json:"tags"`
	Rating       float64        `json:"rating"`
	TotalReviews int32          `json:"totalReviews"`
	Pizzas       []IndexedPizza `json:"pizzas"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}
