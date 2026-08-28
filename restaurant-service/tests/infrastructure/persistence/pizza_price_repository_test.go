package persistence_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

func setupPizzaPriceRepo(t *testing.T) (*gorm.DB, pizza.PizzaPriceRepository, pizza.Pizza, []pizza.PizzaSize) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))
	require.NoError(t, fixtures.LoadPizzaFixtures(t, db.DB))

	p := firstPizza(t, db.DB)

	var sizes []pizza.PizzaSize
	require.NoError(t, db.DB.Order("diameter_cm").Limit(3).Find(&sizes).Error)
	require.Len(t, sizes, 3)

	return db.DB, persistence.NewPizzaPriceRepository(db.DB), p, sizes
}

func mustNewPizzaPrice(t *testing.T, pizzaID, sizeID uuid.UUID, price string) pizza.PizzaPrice {
	t.Helper()

	p, err := pizza.NewPizzaPrice(pizzaID, sizeID, decimal.RequireFromString(price))
	require.NoError(t, err)

	return *p
}

func TestPizzaPriceRepository_ReplacePrices_UpsertsAndDeactivates(t *testing.T) {
	_, repo, p, sizes := setupPizzaPriceRepo(t)

	first := []pizza.PizzaPrice{
		mustNewPizzaPrice(t, p.ID, sizes[0].ID, "9.50"),
		mustNewPizzaPrice(t, p.ID, sizes[1].ID, "12.00"),
	}

	require.NoError(t, repo.ReplacePrices(context.Background(), p.ID, first))

	all, err := repo.ListByPizza(context.Background(), p.ID)
	require.NoError(t, err)
	require.Len(t, all, 2)
	for _, price := range all {
		assert.True(t, price.IsActive)
		assert.NotNil(t, price.UpdatedAt, "UpdatedAt must not be nil after ReplacePrices upsert")
	}

	// second call: drop sizes[0], keep sizes[1] at a new price, add sizes[2]
	second := []pizza.PizzaPrice{
		mustNewPizzaPrice(t, p.ID, sizes[1].ID, "13.50"),
		mustNewPizzaPrice(t, p.ID, sizes[2].ID, "16.00"),
	}

	require.NoError(t, repo.ReplacePrices(context.Background(), p.ID, second))

	all, err = repo.ListByPizza(context.Background(), p.ID)
	require.NoError(t, err)
	require.Len(t, all, 3, "size[0]'s row must still exist, just deactivated, not deleted")

	byID := make(map[uuid.UUID]pizza.PizzaPrice, len(all))
	for _, price := range all {
		byID[price.SizeID] = price
	}

	assert.False(t, byID[sizes[0].ID].IsActive, "dropped size becomes inactive, not deleted")
	assert.NotNil(t, byID[sizes[0].ID].UpdatedAt, "deactivation must set UpdatedAt manually (autoUpdateTime disabled)")
	assert.True(t, byID[sizes[1].ID].IsActive)
	assert.True(t, decimal.RequireFromString("13.50").Equal(byID[sizes[1].ID].Price), "price updated on re-upsert")
	assert.True(t, byID[sizes[2].ID].IsActive)
}

func TestPizzaPriceRepository_ReplacePrices_EmptyDeactivatesAll(t *testing.T) {
	_, repo, p, sizes := setupPizzaPriceRepo(t)

	initial := []pizza.PizzaPrice{
		mustNewPizzaPrice(t, p.ID, sizes[0].ID, "9.50"),
	}
	require.NoError(t, repo.ReplacePrices(context.Background(), p.ID, initial))

	require.NoError(t, repo.ReplacePrices(context.Background(), p.ID, nil))

	all, err := repo.ListByPizza(context.Background(), p.ID)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.False(t, all[0].IsActive)
}

func TestPizzaPriceRepository_WithTx_NestsAsSavepointAndRollsBackWithOuterTx(t *testing.T) {
	db, repo, p, sizes := setupPizzaPriceRepo(t)

	prices := []pizza.PizzaPrice{
		mustNewPizzaPrice(t, p.ID, sizes[0].ID, "9.50"),
	}

	rollbackErr := errors.New("simulated outer transaction failure")

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := repo.WithTx(tx).ReplacePrices(context.Background(), p.ID, prices); err != nil {
			return err
		}
		return rollbackErr
	})

	require.ErrorIs(t, err, rollbackErr)

	all, err := repo.ListByPizza(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Empty(t, all, "ReplacePrices must roll back when the outer transaction fails")
}

func TestPizzaPriceRepository_WithTx_CommitsWithOuterTx(t *testing.T) {
	db, repo, p, sizes := setupPizzaPriceRepo(t)

	prices := []pizza.PizzaPrice{
		mustNewPizzaPrice(t, p.ID, sizes[0].ID, "9.50"),
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		return repo.WithTx(tx).ReplacePrices(context.Background(), p.ID, prices)
	})
	require.NoError(t, err)

	all, err := repo.ListByPizza(context.Background(), p.ID)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.True(t, all[0].IsActive)
}
