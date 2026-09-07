package persistence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/order"
	"order-service/internal/infrastructure/geocoder"
	"order-service/internal/infrastructure/persistence"
	"order-service/tests/testutil"
)

func setupGeocodeRepo(t *testing.T) order.GeocodeRepository {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableGeocode)

	return persistence.NewGeocodeRepository(db.DB)
}

func TestGeocodeRepository_FindByHash_NotFound(t *testing.T) {
	repo := setupGeocodeRepo(t)

	found, err := repo.FindByHash(context.Background(), "does-not-exist")

	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestGeocodeRepository_CreateAndFindByHash(t *testing.T) {
	repo := setupGeocodeRepo(t)

	entry := order.GeocodeEntry{AddressHash: "hash-1", Lat: 53.5511, Lon: 9.9937}
	require.NoError(t, repo.Create(context.Background(), entry))

	found, err := repo.FindByHash(context.Background(), "hash-1")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.InDelta(t, 53.5511, found.Lat, 0.0001)
	assert.InDelta(t, 9.9937, found.Lon, 0.0001)
	assert.False(t, found.CreatedAt.IsZero())
}

func TestGeocodeRepository_Create_DuplicateHashIsNoOp(t *testing.T) {
	repo := setupGeocodeRepo(t)

	original := order.GeocodeEntry{AddressHash: "hash-1", Lat: 53.5511, Lon: 9.9937}
	require.NoError(t, repo.Create(context.Background(), original))

	duplicate := order.GeocodeEntry{AddressHash: "hash-1", Lat: 0, Lon: 0}
	require.NoError(t, repo.Create(context.Background(), duplicate))

	found, err := repo.FindByHash(context.Background(), "hash-1")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.InDelta(t, 53.5511, found.Lat, 0.0001, "the first writer's coordinates must win, not be overwritten")
}

type fakeInnerGeocoder struct {
	lat, lon float64
	err      error
	calls    int
}

func (f *fakeInnerGeocoder) Geocode(_ context.Context, _ order.Address) (float64, float64, error) {
	f.calls++
	if f.err != nil {
		return 0, 0, f.err
	}

	return f.lat, f.lon, nil
}

func TestCachingGeocoder_CachesAcrossCalls(t *testing.T) {
	repo := setupGeocodeRepo(t)
	inner := &fakeInnerGeocoder{lat: 53.5511, lon: 9.9937}
	g := geocoder.NewCachingGeocoder(repo, inner)

	addr := order.Address{House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345"}

	lat1, lon1, err := g.Geocode(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls)
	assert.InDelta(t, 53.5511, lat1, 0.0001)
	assert.InDelta(t, 9.9937, lon1, 0.0001)

	lat2, lon2, err := g.Geocode(context.Background(), addr)
	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls, "second call for the same address must hit the cache, not the inner geocoder")
	assert.Equal(t, lat1, lat2)
	assert.Equal(t, lon1, lon2)
}

func TestCachingGeocoder_NormalizesAddressForCacheKey(t *testing.T) {
	repo := setupGeocodeRepo(t)
	inner := &fakeInnerGeocoder{lat: 53.5511, lon: 9.9937}
	g := geocoder.NewCachingGeocoder(repo, inner)

	_, _, err := g.Geocode(context.Background(), order.Address{
		House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345",
	})
	require.NoError(t, err)

	_, _, err = g.Geocode(context.Background(), order.Address{
		House: "  1 ", Street: "MAIN   ST", City: "hamburg", PostalCode: "12345",
	})
	require.NoError(t, err)

	assert.Equal(t, 1, inner.calls, "case/whitespace differences must normalize to the same cache key")
}

func TestCachingGeocoder_DifferentAddresses_SeparateCacheEntries(t *testing.T) {
	repo := setupGeocodeRepo(t)
	inner := &fakeInnerGeocoder{lat: 53.5511, lon: 9.9937}
	g := geocoder.NewCachingGeocoder(repo, inner)

	_, _, err := g.Geocode(context.Background(), order.Address{
		House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345",
	})
	require.NoError(t, err)

	_, _, err = g.Geocode(context.Background(), order.Address{
		House: "2", Street: "Other St", City: "Berlin", PostalCode: "10117",
	})
	require.NoError(t, err)

	assert.Equal(t, 2, inner.calls, "distinct addresses must not share a cache entry")
}

func TestCachingGeocoder_InnerError_NotCached(t *testing.T) {
	repo := setupGeocodeRepo(t)
	inner := &fakeInnerGeocoder{err: errors.New("no geocoding results found")}
	g := geocoder.NewCachingGeocoder(repo, inner)

	addr := order.Address{House: "1", Street: "Main St", City: "Hamburg", PostalCode: "12345"}

	_, _, err := g.Geocode(context.Background(), addr)
	require.Error(t, err)
	assert.Equal(t, 1, inner.calls)

	_, _, err = g.Geocode(context.Background(), addr)
	require.Error(t, err)
	assert.Equal(t, 2, inner.calls, "a failed lookup must not be cached — retried on the next call")
}
