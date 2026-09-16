package customer_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"customer-service/internal/domain/customer"
)

func TestNewAddress_SetsFields(t *testing.T) {
	customerID := uuid.New()

	a, err := customer.NewAddress(customerID, "10", "Main St", "Berlin", "10115", true)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, a.ID)
	assert.Equal(t, customerID, a.CustomerID)
	assert.Equal(t, "10", a.House)
	assert.Equal(t, "Main St", a.Street)
	assert.Equal(t, "Berlin", a.City)
	assert.Equal(t, "10115", a.PostalCode)
	assert.True(t, a.IsDefault)
}

func TestNewAddress_NotDefault(t *testing.T) {
	a, err := customer.NewAddress(uuid.New(), "10", "Main St", "Berlin", "10115", false)

	require.NoError(t, err)
	assert.False(t, a.IsDefault)
}
