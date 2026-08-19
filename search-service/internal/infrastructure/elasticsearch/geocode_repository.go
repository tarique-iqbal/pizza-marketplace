package elasticsearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"

	"search-service/internal/domain/index"
)

const GeocodeIndexName = "geocode"

// CachingGeocoder wraps another Geocoder with an Elasticsearch-backed cache,
// keyed by a normalized address. A cache hit costs one ES point lookup
// instead of a paid, rate-limited OpenCage call — this matters because many
// customers plausibly search from the same address (same street, same
// building), and /search is hit far more often than a restaurant's own
// address ever changes.
type CachingGeocoder struct {
	es    *elasticsearch.Client
	inner index.Geocoder
}

func NewCachingGeocoder(es *elasticsearch.Client, inner index.Geocoder) *CachingGeocoder {
	return &CachingGeocoder{es: es, inner: inner}
}

type geocodeCacheEntry struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (g *CachingGeocoder) Geocode(ctx context.Context, addr index.Address) (float64, float64, error) {
	key := addressCacheKey(addr)

	if lat, lon, ok, err := g.lookup(ctx, key); err != nil {
		return 0, 0, err
	} else if ok {
		return lat, lon, nil
	}

	lat, lon, err := g.inner.Geocode(ctx, addr)
	if err != nil {
		return 0, 0, err
	}

	if err := g.store(ctx, key, lat, lon); err != nil {
		return 0, 0, fmt.Errorf("failed to cache geocode result: %w", err)
	}

	return lat, lon, nil
}

func (g *CachingGeocoder) lookup(ctx context.Context, key string) (lat, lon float64, ok bool, err error) {
	res, err := g.es.Get(GeocodeIndexName, key, g.es.Get.WithContext(ctx))
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to query geocode cache: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return 0, 0, false, nil
	}
	if res.IsError() {
		return 0, 0, false, fmt.Errorf("geocode cache lookup failed: %s", res.String())
	}

	var doc struct {
		Source geocodeCacheEntry `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return 0, 0, false, fmt.Errorf("failed to decode geocode cache entry: %w", err)
	}

	return doc.Source.Lat, doc.Source.Lon, true, nil
}

func (g *CachingGeocoder) store(ctx context.Context, key string, lat, lon float64) error {
	body, err := json.Marshal(geocodeCacheEntry{Lat: lat, Lon: lon})
	if err != nil {
		return fmt.Errorf("failed to marshal geocode cache entry: %w", err)
	}

	res, err := g.es.Index(
		GeocodeIndexName,
		bytes.NewReader(body),
		g.es.Index.WithDocumentID(key),
		g.es.Index.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index geocode cache entry: %s", res.String())
	}

	return nil
}

func addressCacheKey(addr index.Address) string {
	normalized := strings.Join([]string{
		normalizeAddressPart(addr.House),
		normalizeAddressPart(addr.Street),
		normalizeAddressPart(addr.City),
		normalizeAddressPart(addr.PostalCode),
	}, "|")

	sum := sha256.Sum256([]byte(normalized))

	return hex.EncodeToString(sum[:])
}

func normalizeAddressPart(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
