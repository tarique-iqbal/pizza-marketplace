package elasticsearch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"search-service/internal/domain/index"
	esinfra "search-service/internal/infrastructure/elasticsearch"
	"search-service/tests/testutil"
)

type fakeInnerGeocoder struct {
	lat, lon float64
	err      error
	calls    int
}

func (f *fakeInnerGeocoder) Geocode(_ context.Context, _ index.Address) (float64, float64, error) {
	f.calls++
	if f.err != nil {
		return 0, 0, f.err
	}
	return f.lat, f.lon, nil
}

func TestCachingGeocoder_CachesAcrossCalls(t *testing.T) {
	es := testutil.ES(t)
	inner := &fakeInnerGeocoder{lat: 53.5511, lon: 9.9937}
	geocoder := esinfra.NewCachingGeocoder(es, inner)

	addr := index.Address{House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345"}

	lat1, lon1, err := geocoder.Geocode(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls)
	assert.InDelta(t, 53.5511, lat1, 0.0001)
	assert.InDelta(t, 9.9937, lon1, 0.0001)

	lat2, lon2, err := geocoder.Geocode(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls, "second call for the same address must hit the cache, not the inner geocoder")
	assert.Equal(t, lat1, lat2)
	assert.Equal(t, lon1, lon2)
}

func TestCachingGeocoder_NormalizesAddressForCacheKey(t *testing.T) {
	es := testutil.ES(t)
	inner := &fakeInnerGeocoder{lat: 53.5511, lon: 9.9937}
	geocoder := esinfra.NewCachingGeocoder(es, inner)

	_, _, err := geocoder.Geocode(context.Background(), index.Address{
		House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345",
	})
	require.NoError(t, err)

	_, _, err = geocoder.Geocode(context.Background(), index.Address{
		House: "  1 ", Street: "MAIN   ST", City: "hamburg", PostalCode: "12345",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, inner.calls, "case/whitespace differences must normalize to the same cache key")
}

func TestCachingGeocoder_DifferentAddresses_SeparateCacheEntries(t *testing.T) {
	es := testutil.ES(t)
	inner := &fakeInnerGeocoder{lat: 53.5511, lon: 9.9937}
	geocoder := esinfra.NewCachingGeocoder(es, inner)

	_, _, err := geocoder.Geocode(context.Background(), index.Address{
		House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345",
	})
	require.NoError(t, err)

	_, _, err = geocoder.Geocode(context.Background(), index.Address{
		House: "2", Street: "Other St", City: "Berlin", PostalCode: "10117",
	})
	require.NoError(t, err)

	assert.Equal(t, 2, inner.calls, "distinct addresses must not share a cache entry")
}

func TestCachingGeocoder_InnerError_NotCached(t *testing.T) {
	es := testutil.ES(t)
	inner := &fakeInnerGeocoder{err: errors.New("no geocoding results found")}
	geocoder := esinfra.NewCachingGeocoder(es, inner)

	addr := index.Address{House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345"}

	_, _, err := geocoder.Geocode(context.Background(), addr)
	require.Error(t, err)
	assert.Equal(t, 1, inner.calls)

	_, _, err = geocoder.Geocode(context.Background(), addr)
	require.Error(t, err)
	assert.Equal(t, 2, inner.calls, "a failed lookup must not be cached — retried on the next call")
}
