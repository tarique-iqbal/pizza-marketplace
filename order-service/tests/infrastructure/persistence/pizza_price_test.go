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

func setupPizzaPriceRepo(t *testing.T) (readmodel.PizzaPriceRepository, []readmodel.PizzaPrice) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	restaurants := fixtures.LoadRestaurantFixtures(t, db.DB)
	pizzas := fixtures.LoadPizzaFixtures(t, db.DB, restaurants[0].ID)
	prices := fixtures.LoadPizzaPriceFixtures(t, db.DB, pizzas[0].ID)

	return persistence.NewPizzaPriceRepository(db.DB), prices
}

func TestPizzaPriceRepository_ListByPizza(t *testing.T) {
	repo, seeded := setupPizzaPriceRepo(t)

	found, err := repo.ListByPizza(context.Background(), seeded[0].PizzaID)

	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestPizzaPriceRepository_Upsert_NewerOverwrites(t *testing.T) {
	repo, seeded := setupPizzaPriceRepo(t)
	original := seeded[0]

	updated := original
	updated.Price = decimal.NewFromFloat(99.99)
	updated.UpdatedAt = original.UpdatedAt.Add(time.Hour)

	err := repo.Upsert(context.Background(), updated)
	require.NoError(t, err)

	found, err := repo.ListByPizza(context.Background(), original.PizzaID)
	require.NoError(t, err)

	for _, p := range found {
		if p.SizeID == original.SizeID {
			assert.True(t, p.Price.Equal(decimal.NewFromFloat(99.99)))
		}
	}
}

func TestPizzaPriceRepository_Upsert_StaleIsNoOp(t *testing.T) {
	repo, seeded := setupPizzaPriceRepo(t)
	original := seeded[0]

	stale := original
	stale.Price = decimal.NewFromFloat(0.01)
	stale.UpdatedAt = original.UpdatedAt.Add(-time.Hour)

	err := repo.Upsert(context.Background(), stale)
	require.NoError(t, err)

	found, err := repo.ListByPizza(context.Background(), original.PizzaID)
	require.NoError(t, err)

	for _, p := range found {
		if p.SizeID == original.SizeID {
			assert.True(t, p.Price.Equal(original.Price))
		}
	}
}
