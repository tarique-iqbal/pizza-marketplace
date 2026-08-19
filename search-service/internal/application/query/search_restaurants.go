package query

import (
	"context"
	"fmt"

	"search-service/internal/domain/index"
)

type SearchRestaurants struct {
	repo     index.SearchRepository
	geocoder index.Geocoder
}

func NewSearchRestaurants(repo index.SearchRepository, geocoder index.Geocoder) *SearchRestaurants {
	return &SearchRestaurants{repo: repo, geocoder: geocoder}
}

func (uc *SearchRestaurants) Execute(
	ctx context.Context,
	address index.Address,
	text string,
) ([]index.IndexedRestaurant, error) {
	lat, lon, err := uc.geocoder.Geocode(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	q := index.SearchQuery{
		Text:     text,
		Location: index.GeoPoint{Lat: lat, Lon: lon},
	}

	return uc.repo.Search(ctx, q)
}
