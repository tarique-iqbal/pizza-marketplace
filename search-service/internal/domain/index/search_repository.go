package index

import (
	"context"

	"github.com/google/uuid"
)

type SearchQuery struct {
	Text     string
	Location GeoPoint
}

type SearchRepository interface {
	UpsertSnapshot(ctx context.Context, r IndexedRestaurant) error
	UpdateFields(ctx context.Context, id uuid.UUID, fields RestaurantFields) error
	Search(ctx context.Context, q SearchQuery) ([]IndexedRestaurant, error)
}
