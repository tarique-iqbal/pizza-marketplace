package order

import "context"

type Geocoder interface {
	Geocode(ctx context.Context, addr Address) (lat, lon float64, err error)
}
