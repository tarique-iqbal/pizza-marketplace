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

type SearchRestaurantsRequest struct {
	Address     index.Address
	Text        string
	Fulfillment string
	Tags        []string
	OpenNow     bool
	Sort        string
}

func (uc *SearchRestaurants) Execute(
	ctx context.Context,
	req SearchRestaurantsRequest,
) ([]index.IndexedRestaurant, error) {
	lat, lon, err := uc.geocoder.Geocode(ctx, req.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	q := index.SearchQuery{
		Text:        req.Text,
		Location:    index.GeoPoint{Lat: lat, Lon: lon},
		Fulfillment: req.Fulfillment,
		Tags:        req.Tags,
		OpenNow:     req.OpenNow,
		Sort:        req.Sort,
	}

	return uc.repo.Search(ctx, q)
}
