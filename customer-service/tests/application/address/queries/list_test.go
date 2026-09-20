package queries_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appqueries "customer-service/internal/application/address/queries"
	"customer-service/internal/domain/customer"
	"customer-service/tests/testutil"
)

func TestList_Execute_ReturnsMappedAddresses(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{
		ListByCustomerResult: []customer.Address{
			{House: "10", Street: "Main St", City: "Berlin", PostalCode: "10115", IsDefault: true},
			{House: "20", Street: "Second St", City: "Berlin", PostalCode: "10117"},
		},
	}
	uc := appqueries.NewList(addressRepo)

	res, err := uc.Execute(context.Background(), testutil.MustNewID())

	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "Main St", res[0].Street)
	assert.True(t, res[0].IsDefault)
	assert.Equal(t, "Second St", res[1].Street)
	assert.False(t, res[1].IsDefault)
}

func TestList_Execute_PropagatesError(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{ListByCustomerErr: assert.AnError}
	uc := appqueries.NewList(addressRepo)

	_, err := uc.Execute(context.Background(), testutil.MustNewID())

	assert.ErrorIs(t, err, assert.AnError)
}
