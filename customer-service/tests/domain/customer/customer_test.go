package customer_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"customer-service/internal/domain/customer"
)

func TestCustomer_SetPhone_SetsField(t *testing.T) {
	c := &customer.Customer{ID: uuid.New()}

	c.SetPhone("+49 40 12345678")

	require.NotNil(t, c.Phone)
	assert.Equal(t, "+49 40 12345678", *c.Phone)
}
