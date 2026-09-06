package geocoder

import (
	"context"
	"fmt"

	"order-service/internal/domain/order"
	"order-service/internal/shared/geo"
)

type CachingGeocoder struct {
	repo  order.GeocodeRepository
	inner order.Geocoder
}

func NewCachingGeocoder(repo order.GeocodeRepository, inner order.Geocoder) *CachingGeocoder {
	return &CachingGeocoder{repo: repo, inner: inner}
}

func (g *CachingGeocoder) Geocode(ctx context.Context, addr order.Address) (float64, float64, error) {
	hash := geo.AddressHash(addr.House, addr.Street, addr.City, addr.PostalCode)

	entry, err := g.repo.FindByHash(ctx, hash)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query geocode cache: %w", err)
	}
	if entry != nil {
		return entry.Lat, entry.Lon, nil
	}

	lat, lon, err := g.inner.Geocode(ctx, addr)
	if err != nil {
		return 0, 0, err
	}

	if err := g.repo.Create(ctx, order.GeocodeEntry{AddressHash: hash, Lat: lat, Lon: lon}); err != nil {
		return 0, 0, fmt.Errorf("failed to cache geocode result: %w", err)
	}

	return lat, lon, nil
}
