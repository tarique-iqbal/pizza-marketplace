package geo

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// AddressHash returns a SHA-256 hex digest of the normalized address, used as the
// geocode cache key. Ported from search-service's addressCacheKey/normalizeAddressPart.
func AddressHash(house, street, city, postalCode string) string {
	normalized := strings.Join([]string{
		normalizeAddressPart(house),
		normalizeAddressPart(street),
		normalizeAddressPart(city),
		normalizeAddressPart(postalCode),
	}, "|")

	sum := sha256.Sum256([]byte(normalized))

	return hex.EncodeToString(sum[:])
}

func normalizeAddressPart(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
