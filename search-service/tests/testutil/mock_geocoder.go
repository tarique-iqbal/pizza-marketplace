package testutil

import (
	"context"

	"search-service/internal/domain/index"
)

type MockGeocoder struct {
	Lat, Lon  float64
	Err       error
	LastAddr  index.Address
	CallCount int
}

var _ index.Geocoder = (*MockGeocoder)(nil)

func (m *MockGeocoder) Geocode(_ context.Context, addr index.Address) (float64, float64, error) {
	m.LastAddr = addr
	m.CallCount++

	if m.Err != nil {
		return 0, 0, m.Err
	}

	return m.Lat, m.Lon, nil
}
