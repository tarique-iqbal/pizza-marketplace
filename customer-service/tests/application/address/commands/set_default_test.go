package commands_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcommands "customer-service/internal/application/address/commands"
	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/persistence"
	apperr "customer-service/internal/shared/errors"
	"customer-service/tests/infrastructure/db/fixtures"
	"customer-service/tests/testutil"
)

func setupSetDefault(t *testing.T) (*appcommands.SetDefault, customer.Customer, []customer.Address) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer)

	fixtureCustomers := fixtures.LoadCustomerFixtures(t, db.DB)
	owner := fixtureCustomers[0]
	fixtureAddresses := fixtures.LoadAddressFixtures(t, db.DB, owner.ID)

	addressRepo := persistence.NewAddressRepository(db.DB)
	uc := appcommands.NewSetDefault(db.DB, addressRepo)

	return uc, owner, fixtureAddresses
}

func TestSetDefault_Execute_SwapsDefault(t *testing.T) {
	db := testutil.DB(t)
	uc, owner, fixtureAddresses := setupSetDefault(t)

	newDefault := fixtureAddresses[1]
	require.False(t, newDefault.IsDefault)

	res, err := uc.Execute(context.Background(), owner.ID, newDefault.ID)
	require.NoError(t, err)
	assert.True(t, res.IsDefault)
	assert.Equal(t, newDefault.ID, res.ID)

	var oldDefault, found customer.Address
	require.NoError(t, db.DB.First(&oldDefault, "id = ?", fixtureAddresses[0].ID).Error)
	require.NoError(t, db.DB.First(&found, "id = ?", newDefault.ID).Error)

	assert.False(t, oldDefault.IsDefault)
	assert.True(t, found.IsDefault)
}

func TestSetDefault_Execute_NotOwned_ReturnsNotFound(t *testing.T) {
	uc, _, fixtureAddresses := setupSetDefault(t)

	_, err := uc.Execute(context.Background(), uuid.New(), fixtureAddresses[1].ID)

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestSetDefault_Execute_UnknownID_ReturnsNotFound(t *testing.T) {
	uc, owner, _ := setupSetDefault(t)

	_, err := uc.Execute(context.Background(), owner.ID, uuid.New())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
