package persistence_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/persistence"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/infrastructure/db/fixtures"
	"order-service/tests/testutil"
)

func setupCustomerRepo(t *testing.T) (readmodel.CustomerRepository, []readmodel.Customer) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomer)

	seeded := fixtures.LoadCustomerFixtures(t, db.DB)

	return persistence.NewCustomerRepository(db.DB), seeded
}

func TestCustomerRepository_FindByID(t *testing.T) {
	repo, seeded := setupCustomerRepo(t)

	found, err := repo.FindByID(context.Background(), seeded[0].ID)

	require.NoError(t, err)
	assert.Equal(t, seeded[0].Email, found.Email)
}

func TestCustomerRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupCustomerRepo(t)

	_, err := repo.FindByID(context.Background(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestCustomerRepository_Upsert_Insert(t *testing.T) {
	repo, _ := setupCustomerRepo(t)

	customer := readmodel.Customer{
		ID:        testutil.MustNewID(),
		Email:     "new.customer@example.com",
		FirstName: "Nina",
	}

	err := repo.Upsert(context.Background(), customer)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), customer.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nina", found.FirstName)
}

func TestCustomerRepository_Upsert_RedeliveryIsNoOp(t *testing.T) {
	repo, seeded := setupCustomerRepo(t)
	original := seeded[0]

	duplicate := original
	duplicate.FirstName = "Should Not Apply"

	err := repo.Upsert(context.Background(), duplicate)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), original.ID)
	require.NoError(t, err)
	assert.Equal(t, original.FirstName, found.FirstName)
}
