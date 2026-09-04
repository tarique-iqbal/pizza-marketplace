package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/persistence"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/infrastructure/db/fixtures"
	"order-service/tests/testutil"
)

func setupPizzaRepo(t *testing.T) (readmodel.PizzaRepository, []readmodel.Pizza) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableRestaurant)

	restaurants := fixtures.LoadRestaurantFixtures(t, db.DB)
	pizzas := fixtures.LoadPizzaFixtures(t, db.DB, restaurants[0].ID)

	return persistence.NewPizzaRepository(db.DB), pizzas
}

func TestPizzaRepository_FindByID(t *testing.T) {
	repo, seeded := setupPizzaRepo(t)

	found, err := repo.FindByID(context.Background(), seeded[0].ID)

	require.NoError(t, err)
	assert.Equal(t, seeded[0].Name, found.Name)
}

func TestPizzaRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupPizzaRepo(t)

	_, err := repo.FindByID(context.Background(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestPizzaRepository_Upsert_StaleIsNoOp(t *testing.T) {
	repo, seeded := setupPizzaRepo(t)
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

func TestPizzaRepository_Delete(t *testing.T) {
	repo, seeded := setupPizzaRepo(t)
	original := seeded[0]

	err := repo.Delete(context.Background(), original.ID, original.UpdatedAt.Add(time.Hour))
	require.NoError(t, err)

	_, err = repo.FindByID(context.Background(), original.ID)
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestPizzaRepository_Delete_StaleIsNoOp(t *testing.T) {
	repo, seeded := setupPizzaRepo(t)
	original := seeded[0]

	err := repo.Delete(context.Background(), original.ID, original.UpdatedAt.Add(-time.Hour))
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), original.ID)
	require.NoError(t, err)
	assert.Equal(t, original.Name, found.Name)
}
