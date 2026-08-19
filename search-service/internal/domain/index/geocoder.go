package index

import "context"

type Address struct {
	House      string
	Street     string
	City       string
	PostalCode string
}

type Geocoder interface {
	Geocode(ctx context.Context, addr Address) (lat, lon float64, err error)
}
