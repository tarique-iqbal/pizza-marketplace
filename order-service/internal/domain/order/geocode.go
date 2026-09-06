package order

import "context"

type GeocodeEntry struct {
	AddressHash string
	Lat         float64
	Lon         float64
}

type GeocodeRepository interface {
	FindByHash(ctx context.Context, hash string) (*GeocodeEntry, error)
	Create(ctx context.Context, entry GeocodeEntry) error
}
