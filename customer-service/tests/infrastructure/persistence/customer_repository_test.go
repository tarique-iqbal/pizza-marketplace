package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/persistence"
	"customer-service/tests/infrastructure/db/fixtures"
	"customer-service/tests/testutil"
)

func setupCustomerRepo(t *testing.T) (customer.CustomerRepository, []customer.Customer) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomer)

	fixtureCustomers := fixtures.LoadCustomerFixtures(t, db.DB)

	return persistence.NewCustomerRepository(db.DB), fixtureCustomers
}

func TestCustomerRepository_Upsert_CreatesNewRow(t *testing.T) {
	db := testutil.DB(t)
	repo, _ := setupCustomerRepo(t)

	c := customer.Customer{
		ID:        testutil.MustNewID(),
		Email:     "new@example.com",
		FirstName: "New",
		LastName:  "Customer",
	}

	err := repo.Upsert(context.Background(), c)
	require.NoError(t, err)

	var found customer.Customer
	err = db.DB.First(&found, "id = ?", c.ID).Error
	require.NoError(t, err)
	assert.Equal(t, c.Email, found.Email)
}

func TestCustomerRepository_Upsert_IgnoresDuplicateID(t *testing.T) {
	db := testutil.DB(t)
	repo, fixtureCustomers := setupCustomerRepo(t)

	existing := fixtureCustomers[0]

	err := repo.Upsert(context.Background(), customer.Customer{
		ID:        existing.ID,
		Email:     "changed@example.com",
		FirstName: "Changed",
		LastName:  "Name",
	})
	require.NoError(t, err)

	var found customer.Customer
	err = db.DB.First(&found, "id = ?", existing.ID).Error
	require.NoError(t, err)
	assert.Equal(t, existing.Email, found.Email)
}

func TestCustomerRepository_Update_SetsPhone(t *testing.T) {
	db := testutil.DB(t)
	repo, fixtureCustomers := setupCustomerRepo(t)

	c := fixtureCustomers[0]
	c.SetPhone("+49 30 987654")

	err := repo.Update(context.Background(), &c)
	require.NoError(t, err)

	var found customer.Customer
	err = db.DB.First(&found, "id = ?", c.ID).Error
	require.NoError(t, err)
	require.NotNil(t, found.Phone)
	assert.Equal(t, "+49 30 987654", *found.Phone)
}

func TestCustomerRepository_FindByID_Found(t *testing.T) {
	repo, fixtureCustomers := setupCustomerRepo(t)

	target := fixtureCustomers[1]

	found, err := repo.FindByID(context.Background(), target.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, target.Email, found.Email)
}

func TestCustomerRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupCustomerRepo(t)

	found, err := repo.FindByID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}
