package persistence_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/persistence"
	apperr "customer-service/internal/shared/errors"
	"customer-service/tests/infrastructure/db/fixtures"
	"customer-service/tests/testutil"
)

func setupAddressRepo(t *testing.T) (customer.AddressRepository, customer.Customer, []customer.Address) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer)

	fixtureCustomers := fixtures.LoadCustomerFixtures(t, db.DB)
	owner := fixtureCustomers[0]
	fixtureAddresses := fixtures.LoadAddressFixtures(t, db.DB, owner.ID)

	return persistence.NewAddressRepository(db.DB), owner, fixtureAddresses
}

func TestAddressRepository_Create(t *testing.T) {
	db := testutil.DB(t)
	repo, owner, _ := setupAddressRepo(t)

	a, err := customer.NewAddress(owner.ID, "30", "Third St", "Berlin", "10119", false)
	require.NoError(t, err)

	err = repo.Create(context.Background(), a)
	require.NoError(t, err)

	var found customer.Address
	err = db.DB.First(&found, "id = ?", a.ID).Error
	require.NoError(t, err)
	assert.Equal(t, owner.ID, found.CustomerID)
	assert.NotZero(t, found.CreatedAt)
}

func TestAddressRepository_Delete_RemovesOwnedRow(t *testing.T) {
	db := testutil.DB(t)
	repo, owner, fixtureAddresses := setupAddressRepo(t)

	target := fixtureAddresses[0]

	err := repo.Delete(context.Background(), target.ID, owner.ID)
	require.NoError(t, err)

	err = db.DB.First(&customer.Address{}, "id = ?", target.ID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAddressRepository_Delete_NotOwned_ReturnsNotFound(t *testing.T) {
	repo, _, fixtureAddresses := setupAddressRepo(t)

	target := fixtureAddresses[0]

	err := repo.Delete(context.Background(), target.ID, uuid.New())
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestAddressRepository_FindByID_Found(t *testing.T) {
	repo, owner, fixtureAddresses := setupAddressRepo(t)

	target := fixtureAddresses[0]

	found, err := repo.FindByID(context.Background(), target.ID, owner.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, target.Street, found.Street)
}

func TestAddressRepository_FindByID_NotOwned_ReturnsNil(t *testing.T) {
	repo, _, fixtureAddresses := setupAddressRepo(t)

	target := fixtureAddresses[0]

	found, err := repo.FindByID(context.Background(), target.ID, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestAddressRepository_ListByCustomer_DefaultFirst(t *testing.T) {
	repo, owner, fixtureAddresses := setupAddressRepo(t)

	list, err := repo.ListByCustomer(context.Background(), owner.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.True(t, list[0].IsDefault)
	assert.Equal(t, fixtureAddresses[0].ID, list[0].ID)
}

func TestAddressRepository_UnsetDefault_ClearsCurrentDefault(t *testing.T) {
	db := testutil.DB(t)
	repo, owner, fixtureAddresses := setupAddressRepo(t)

	err := repo.UnsetDefault(context.Background(), owner.ID)
	require.NoError(t, err)

	var found customer.Address
	err = db.DB.First(&found, "id = ?", fixtureAddresses[0].ID).Error
	require.NoError(t, err)
	assert.False(t, found.IsDefault)
}

func TestAddressRepository_SetDefault_MarksRowDefault(t *testing.T) {
	db := testutil.DB(t)
	repo, owner, fixtureAddresses := setupAddressRepo(t)

	target := fixtureAddresses[1]
	require.False(t, target.IsDefault)

	// clear the fixture's existing default first: the unique index allows only one.
	require.NoError(t, repo.UnsetDefault(context.Background(), owner.ID))

	err := repo.SetDefault(context.Background(), target.ID)
	require.NoError(t, err)

	var found customer.Address
	err = db.DB.First(&found, "id = ?", target.ID).Error
	require.NoError(t, err)
	assert.True(t, found.IsDefault)
}

func TestAddressRepository_SetDefault_UnknownID_ReturnsNotFound(t *testing.T) {
	repo, _, _ := setupAddressRepo(t)

	err := repo.SetDefault(context.Background(), uuid.New())
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestAddressRepository_WithTx_UnsetThenSetDefault(t *testing.T) {
	db := testutil.DB(t)
	repo, owner, fixtureAddresses := setupAddressRepo(t)

	newDefault := fixtureAddresses[1]

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		txRepo := repo.WithTx(tx)
		if err := txRepo.UnsetDefault(context.Background(), owner.ID); err != nil {
			return err
		}

		return txRepo.SetDefault(context.Background(), newDefault.ID)
	})
	require.NoError(t, err)

	var oldDefault, found customer.Address
	require.NoError(t, db.DB.First(&oldDefault, "id = ?", fixtureAddresses[0].ID).Error)
	require.NoError(t, db.DB.First(&found, "id = ?", newDefault.ID).Error)

	assert.False(t, oldDefault.IsDefault)
	assert.True(t, found.IsDefault)
}
