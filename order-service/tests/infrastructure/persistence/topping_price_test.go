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
	"order-service/tests/infrastructure/db/fixtures"
	"order-service/tests/testutil"
)

func setupToppingPriceRepo(t *testing.T) (readmodel.ToppingPriceRepository, []readmodel.ToppingPrice) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	restaurants := fixtures.LoadRestaurantFixtures(t, db.DB)
	prices := fixtures.LoadToppingPriceFixtures(t, db.DB, restaurants[0].ID)

	return persistence.NewToppingPriceRepository(db.DB), prices
}

func TestToppingPriceRepository_ListByRestaurant(t *testing.T) {
	repo, seeded := setupToppingPriceRepo(t)

	found, err := repo.ListByRestaurant(context.Background(), seeded[0].RestaurantID)

	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestToppingPriceRepository_Upsert_NewerOverwrites(t *testing.T) {
	repo, seeded := setupToppingPriceRepo(t)
	original := seeded[0]

	updated := original
	updated.ExtraPrice = decimal.NewFromFloat(2.99)
	updated.UpdatedAt = original.UpdatedAt.Add(time.Hour)

	err := repo.Upsert(context.Background(), updated)
	require.NoError(t, err)

	found, err := repo.ListByRestaurant(context.Background(), original.RestaurantID)
	require.NoError(t, err)

	for _, p := range found {
		if p.ToppingID == original.ToppingID {
			assert.True(t, p.ExtraPrice.Equal(decimal.NewFromFloat(2.99)))
		}
	}
}

func TestToppingPriceRepository_Upsert_StaleIsNoOp(t *testing.T) {
	repo, seeded := setupToppingPriceRepo(t)
	original := seeded[0]

	stale := original
	stale.ExtraPrice = decimal.NewFromFloat(0.01)
	stale.UpdatedAt = original.UpdatedAt.Add(-time.Hour)

	err := repo.Upsert(context.Background(), stale)
	require.NoError(t, err)

	found, err := repo.ListByRestaurant(context.Background(), original.RestaurantID)
	require.NoError(t, err)

	for _, p := range found {
		if p.ToppingID == original.ToppingID {
			assert.True(t, p.ExtraPrice.Equal(original.ExtraPrice))
		}
	}
}
