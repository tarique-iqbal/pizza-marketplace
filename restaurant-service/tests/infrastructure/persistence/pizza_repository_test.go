package persistence_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/pizza"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

type pizzaRepoSetup struct {
	DB        *gorm.DB
	PizzaRepo pizza.PizzaRepository
}

func setupPizzaRepo(t *testing.T) pizzaRepoSetup {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	require.NoError(t, fixtures.LoadRestaurantFixtures(t, db.DB))
	require.NoError(t, fixtures.LoadPizzaFixtures(t, db.DB))

	return pizzaRepoSetup{
		DB:        db.DB,
		PizzaRepo: persistence.NewPizzaRepository(db.DB),
	}
}

func firstPizza(t *testing.T, db *gorm.DB) pizza.Pizza {
	var p pizza.Pizza

	err := db.Order("sort_order").First(&p).Error
	require.NoError(t, err)

	return p
}

func TestPizzaRepository_Create(t *testing.T) {
	setup := setupPizzaRepo(t)

	var owner restaurant.Restaurant
	require.NoError(t, setup.DB.Where("slug = ?", "anatolische-kueche").Take(&owner).Error)

	p := pizza.NewPizza(testutil.MustNewID(), owner.ID).WithDetails(
		"Quattro Formaggi",
		nil,
		true,
		pizza.PizzaAvailable,
		3,
	)

	err := setup.PizzaRepo.Create(context.Background(), p)
	require.NoError(t, err)

	var stored pizza.Pizza
	require.NoError(t, setup.DB.Take(&stored, "id = ?", p.ID).Error)
	assert.Equal(t, "Quattro Formaggi", stored.Name)
	assert.Nil(t, stored.UpdatedAt)
	assert.Equal(t, pizza.PizzaAvailable, stored.Status)
}

func TestPizzaRepository_Update(t *testing.T) {
	setup := setupPizzaRepo(t)

	p := firstPizza(t, setup.DB)

	p.WithDetails(
		p.Name,
		p.Image,
		p.IsVegetarian,
		pizza.PizzaArchived,
		p.SortOrder,
	)

	err := setup.PizzaRepo.Update(context.Background(), &p)
	require.NoError(t, err)

	var updated pizza.Pizza
	require.NoError(t, setup.DB.Take(&updated, "id = ?", p.ID).Error)
	assert.Equal(t, pizza.PizzaArchived, updated.Status)
	assert.NotNil(t, updated.UpdatedAt)
}

func TestPizzaRepository_FindByID(t *testing.T) {
	setup := setupPizzaRepo(t)

	existing := firstPizza(t, setup.DB)

	found, err := setup.PizzaRepo.FindByID(context.Background(), existing.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, existing.Name, found.Name)

	notFound, err := setup.PizzaRepo.FindByID(context.Background(), testutil.MustNewID())
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestPizzaRepository_FindByIDAndRestaurant(t *testing.T) {
	setup := setupPizzaRepo(t)

	existing := firstPizza(t, setup.DB)

	found, err := setup.PizzaRepo.FindByIDAndRestaurant(context.Background(), existing.ID, existing.RestaurantID)
	require.NoError(t, err)
	require.NotNil(t, found)

	notFound, err := setup.PizzaRepo.FindByIDAndRestaurant(context.Background(), existing.ID, testutil.MustNewID())
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestPizzaRepository_ListByRestaurant(t *testing.T) {
	setup := setupPizzaRepo(t)

	existing := firstPizza(t, setup.DB)

	pizzas, err := setup.PizzaRepo.ListByRestaurant(context.Background(), existing.RestaurantID)
	require.NoError(t, err)
	require.Len(t, pizzas, 2)
	assert.Equal(t, "Margherita", pizzas[0].Name)
	assert.Equal(t, "Salami", pizzas[1].Name)
}
