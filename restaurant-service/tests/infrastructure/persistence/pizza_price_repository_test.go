package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

func setupPizzaPriceRepo(t *testing.T) (*gorm.DB, restaurant.PizzaPriceRepository, restaurant.Pizza, []restaurant.PizzaSize) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))
	require.NoError(t, fixtures.LoadPizzaFixtures(t, db.DB))

	pizza := firstPizza(t, db.DB)

	var sizes []restaurant.PizzaSize
	require.NoError(t, db.DB.Order("diameter_cm").Limit(3).Find(&sizes).Error)
	require.Len(t, sizes, 3)

	return db.DB, persistence.NewPizzaPriceRepository(db.DB), pizza, sizes
}

func mustNewPizzaPrice(t *testing.T, pizzaID, sizeID uuid.UUID, price string) restaurant.PizzaPrice {
	t.Helper()

	p, err := restaurant.NewPizzaPrice(pizzaID, sizeID, decimal.RequireFromString(price))
	require.NoError(t, err)

	return *p
}

func TestPizzaPriceRepository_ReplacePrices_UpsertsAndDeactivates(t *testing.T) {
	_, repo, pizza, sizes := setupPizzaPriceRepo(t)

	first := []restaurant.PizzaPrice{
		mustNewPizzaPrice(t, pizza.ID, sizes[0].ID, "9.50"),
		mustNewPizzaPrice(t, pizza.ID, sizes[1].ID, "12.00"),
	}

	require.NoError(t, repo.ReplacePrices(context.Background(), pizza.ID, first))

	all, err := repo.ListByPizza(context.Background(), pizza.ID)
	require.NoError(t, err)
	require.Len(t, all, 2)
	for _, p := range all {
		assert.True(t, p.IsActive)
		assert.NotNil(t, p.UpdatedAt, "UpdatedAt must not be nil after ReplacePrices upsert")
	}

	// second call: drop sizes[0], keep sizes[1] at a new price, add sizes[2]
	second := []restaurant.PizzaPrice{
		mustNewPizzaPrice(t, pizza.ID, sizes[1].ID, "13.50"),
		mustNewPizzaPrice(t, pizza.ID, sizes[2].ID, "16.00"),
	}

	require.NoError(t, repo.ReplacePrices(context.Background(), pizza.ID, second))

	all, err = repo.ListByPizza(context.Background(), pizza.ID)
	require.NoError(t, err)
	require.Len(t, all, 3, "size[0]'s row must still exist, just deactivated, not deleted")

	byID := make(map[uuid.UUID]restaurant.PizzaPrice, len(all))
	for _, p := range all {
		byID[p.SizeID] = p
	}

	assert.False(t, byID[sizes[0].ID].IsActive, "dropped size becomes inactive, not deleted")
	assert.NotNil(t, byID[sizes[0].ID].UpdatedAt, "deactivation must explicitly set UpdatedAt now that GORM's autoUpdateTime is disabled")
	assert.True(t, byID[sizes[1].ID].IsActive)
	assert.True(t, decimal.RequireFromString("13.50").Equal(byID[sizes[1].ID].Price), "price updated on re-upsert")
	assert.True(t, byID[sizes[2].ID].IsActive)
}

func TestPizzaPriceRepository_ReplacePrices_EmptyDeactivatesAll(t *testing.T) {
	_, repo, pizza, sizes := setupPizzaPriceRepo(t)

	initial := []restaurant.PizzaPrice{
		mustNewPizzaPrice(t, pizza.ID, sizes[0].ID, "9.50"),
	}
	require.NoError(t, repo.ReplacePrices(context.Background(), pizza.ID, initial))

	require.NoError(t, repo.ReplacePrices(context.Background(), pizza.ID, nil))

	all, err := repo.ListByPizza(context.Background(), pizza.ID)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.False(t, all[0].IsActive)
}
