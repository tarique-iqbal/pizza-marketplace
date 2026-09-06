package geo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"order-service/internal/shared/geo"
)

func TestHaversineKm_SamePoint_ReturnsZero(t *testing.T) {
	d := geo.HaversineKm(53.5511, 9.9937, 53.5511, 9.9937)
	assert.InDelta(t, 0, d, 0.0001)
}

func TestHaversineKm_KnownDistance_HamburgToBerlin(t *testing.T) {
	d := geo.HaversineKm(53.5511, 9.9937, 52.5200, 13.4050)
	assert.InDelta(t, 255, d, 5)
}

func TestHaversineKm_Symmetric(t *testing.T) {
	d1 := geo.HaversineKm(53.5511, 9.9937, 52.5200, 13.4050)
	d2 := geo.HaversineKm(52.5200, 13.4050, 53.5511, 9.9937)
	assert.InDelta(t, d1, d2, 0.0001)
}
