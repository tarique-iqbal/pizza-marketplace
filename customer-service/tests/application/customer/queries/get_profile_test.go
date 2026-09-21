package queries_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appqueries "customer-service/internal/application/customer/queries"
	"customer-service/internal/domain/customer"
	apperr "customer-service/internal/shared/errors"
	"customer-service/tests/testutil"
)

func TestGetProfile_Execute_Found(t *testing.T) {
	id := testutil.MustNewID()
	phone := "+49 30 1234567"
	customerRepo := &testutil.MockCustomerRepository{
		FindByIDResult: &customer.Customer{
			ID:        id,
			Email:     "jane@example.com",
			FirstName: "Jane",
			LastName:  "Doe",
			Phone:     &phone,
		},
	}
	qry := appqueries.NewGetProfile(customerRepo)

	res, err := qry.Execute(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, id, res.ID)
	assert.Equal(t, "jane@example.com", res.Email)
	assert.Equal(t, "Jane", res.FirstName)
	assert.Equal(t, "Doe", res.LastName)
	require.NotNil(t, res.Phone)
	assert.Equal(t, phone, *res.Phone)
}

func TestGetProfile_Execute_NotFound(t *testing.T) {
	customerRepo := &testutil.MockCustomerRepository{}
	qry := appqueries.NewGetProfile(customerRepo)

	_, err := qry.Execute(context.Background(), testutil.MustNewID())

	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
