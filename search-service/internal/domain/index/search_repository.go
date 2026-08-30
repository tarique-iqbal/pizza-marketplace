package index

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SearchQuery struct {
	Text        string
	Location    GeoPoint
	Fulfillment string
	Tags        []string
	OpenNow     bool
	Sort        string
}

type SearchRepository interface {
	UpsertSnapshot(ctx context.Context, r IndexedRestaurant) error
	UpdateFields(ctx context.Context, id uuid.UUID, fields RestaurantFields) error
	UpsertPizza(ctx context.Context, restaurantID uuid.UUID, pizza IndexedPizza) error
	RemovePizza(ctx context.Context, restaurantID, pizzaID uuid.UUID, updatedAt time.Time) error
	UpdateToppingPrices(ctx context.Context, restaurantID uuid.UUID, prices []IndexedToppingPrice, updatedAt time.Time) error
	Search(ctx context.Context, q SearchQuery) ([]IndexedRestaurant, error)
}
