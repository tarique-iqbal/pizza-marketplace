package index

import (
	"time"

	"github.com/google/uuid"
)

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type IndexedPizzaPrice struct {
	SizeID     uuid.UUID `json:"sizeId"`
	DiameterCm int16     `json:"diameterCm"`
	Price      string    `json:"price"`
}

type IndexedPizza struct {
	ID           uuid.UUID           `json:"id"`
	Name         string              `json:"name"`
	IsVegetarian bool                `json:"isVegetarian"`
	Toppings     []string            `json:"toppings"`
	Prices       []IndexedPizzaPrice `json:"prices"`
	UpdatedAt    time.Time           `json:"updatedAt"`
}

type IndexedToppingPrice struct {
	ToppingID  uuid.UUID `json:"toppingId"`
	Name       string    `json:"name"`
	ExtraPrice string    `json:"extraPrice"`
}

type IndexedRestaurant struct {
	ID                     uuid.UUID             `json:"id"`
	Name                   string                `json:"name"`
	Slug                   string                `json:"slug"`
	City                   string                `json:"city"`
	Location               GeoPoint              `json:"location"`
	Currency               string                `json:"currency"`
	Pickup                 bool                  `json:"pickup"`
	DeliveryType           string                `json:"deliveryType"`
	DeliveryKm             *int16                `json:"deliveryKm,omitempty"`
	Tags                   []string              `json:"tags"`
	Rating                 float64               `json:"rating"`
	TotalReviews           int32                 `json:"totalReviews"`
	Pizzas                 []IndexedPizza        `json:"pizzas"`
	ToppingPrices          []IndexedToppingPrice `json:"toppingPrices"`
	UpdatedAt              time.Time             `json:"updatedAt"`
	ToppingPricesUpdatedAt time.Time             `json:"toppingPricesUpdatedAt"`
}

type RestaurantFields struct {
	Name         string
	Slug         string
	City         string
	Location     GeoPoint
	Currency     string
	Pickup       bool
	DeliveryType string
	DeliveryKm   *int16
	Tags         []string
	Rating       float64
	TotalReviews int32
	UpdatedAt    time.Time
}
