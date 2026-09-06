package geo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"order-service/internal/shared/geo"
)

func TestAddressHash_NormalizesCaseAndWhitespace(t *testing.T) {
	h1 := geo.AddressHash("1", "Main St", "Hamburg", "12345")
	h2 := geo.AddressHash("  1 ", "MAIN   ST", "hamburg", "12345")

	assert.Equal(t, h1, h2, "case/whitespace differences must normalize to the same hash")
}

func TestAddressHash_DifferentAddresses_ProduceDifferentHashes(t *testing.T) {
	h1 := geo.AddressHash("1", "Main St", "Hamburg", "12345")
	h2 := geo.AddressHash("2", "Other St", "Berlin", "10117")

	assert.NotEqual(t, h1, h2)
}

func TestAddressHash_IsA64CharHexDigest(t *testing.T) {
	h := geo.AddressHash("1", "Main St", "Hamburg", "12345")

	assert.Len(t, h, 64)
	assert.Regexp(t, "^[0-9a-f]{64}$", h)
}
