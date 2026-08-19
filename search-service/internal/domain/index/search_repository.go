package index

import "context"

type SearchQuery struct {
	Text     string
	Location GeoPoint
}

type SearchRepository interface {
	UpsertSnapshot(ctx context.Context, r IndexedRestaurant) error
	Search(ctx context.Context, q SearchQuery) ([]IndexedRestaurant, error)
}
