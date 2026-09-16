package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apphandlers "customer-service/internal/application/customer/handlers"
	"customer-service/internal/domain/customer"
	"customer-service/tests/testutil"
)

func newUserRegisteredPayload(t *testing.T) customer.EventPayload {
	body := map[string]any{
		"user_id":    testutil.MustNewID().String(),
		"email":      "john.doe@example.com",
		"first_name": "John",
		"last_name":  "Doe",
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return customer.EventPayload{Name: "user.registered", Data: data}
}

func TestUserRegistered_Handle_UpsertsCustomer(t *testing.T) {
	customerRepo := &testutil.MockCustomerRepository{}
	h := apphandlers.NewUserRegistered(customerRepo)

	err := h.Handle(newUserRegisteredPayload(t))

	require.NoError(t, err)
	require.Len(t, customerRepo.Upserted, 1)
	assert.Equal(t, "john.doe@example.com", customerRepo.Upserted[0].Email)
	assert.Equal(t, "John", customerRepo.Upserted[0].FirstName)
	assert.Equal(t, "Doe", customerRepo.Upserted[0].LastName)
}

func TestUserRegistered_Handle_PropagatesUpsertError(t *testing.T) {
	customerRepo := &testutil.MockCustomerRepository{UpsertErr: assert.AnError}
	h := apphandlers.NewUserRegistered(customerRepo)

	err := h.Handle(newUserRegisteredPayload(t))

	assert.ErrorIs(t, err, assert.AnError)
}
