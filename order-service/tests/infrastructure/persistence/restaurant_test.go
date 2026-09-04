package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/persistence"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/infrastructure/db/fixtures"
	"order-service/tests/testutil"
)

func setupRestaurantRepo(t *testing.T) (readmodel.RestaurantRepository, []readmodel.Restaurant) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	seeded := fixtures.LoadRestaurantFixtures(t, db.DB)

	return persistence.NewRestaurantRepository(db.DB), seeded
}

func TestRestaurantRepository_FindByID(t *testing.T) {
	repo, seeded := setupRestaurantRepo(t)

	found, err := repo.FindByID(context.Background(), seeded[0].ID)

	require.NoError(t, err)
	assert.Equal(t, seeded[0].Name, found.Name)
	assert.Equal(t, seeded[0].OwnerEmail, found.OwnerEmail)
}

func TestRestaurantRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupRestaurantRepo(t)

	_, err := repo.FindByID(context.Background(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestRestaurantRepository_Upsert_Insert(t *testing.T) {
	repo, _ := setupRestaurantRepo(t)

	deliveryKm := int16(5)
	restaurant := readmodel.Restaurant{
		ID:           testutil.MustNewID(),
		OwnerID:      testutil.MustNewID(),
		Name:         "New Slice",
		OwnerEmail:   "owner@newslice.de",
		Lat:          48.1351,
		Lon:          11.5820,
		DeliveryKm:   &deliveryKm,
		DeliveryFee:  decimal.NewFromFloat(3.00),
		MinimumOrder: decimal.NewFromFloat(12.00),
		Pickup:       true,
		DeliveryType: readmodel.DeliveryOwn,
		Currency:     "EUR",
		UpdatedAt:    time.Now().UTC(),
	}

	err := repo.Upsert(context.Background(), restaurant)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), restaurant.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Slice", found.Name)
}

func TestRestaurantRepository_Upsert_NewerOverwrites(t *testing.T) {
	repo, seeded := setupRestaurantRepo(t)
	original := seeded[0]

	updated := original
	updated.Name = "Pizza Paradise Updated"
	updated.UpdatedAt = original.UpdatedAt.Add(time.Hour)

	err := repo.Upsert(context.Background(), updated)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), original.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pizza Paradise Updated", found.Name)
}

func TestRestaurantRepository_Upsert_StaleIsNoOp(t *testing.T) {
	repo, seeded := setupRestaurantRepo(t)
	original := seeded[0]

	stale := original
	stale.Name = "Should Not Apply"
	stale.UpdatedAt = original.UpdatedAt.Add(-time.Hour)

	err := repo.Upsert(context.Background(), stale)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), original.ID)
	require.NoError(t, err)
	assert.Equal(t, original.Name, found.Name)
}
