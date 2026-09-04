package readmodel_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreadmodel "order-service/internal/application/readmodel"
	"order-service/internal/domain/readmodel"
	"order-service/tests/testutil"
)

func newUserRegisteredPayload(t *testing.T, role string) readmodel.EventPayload {
	body := map[string]any{
		"user_id":    testutil.MustNewID().String(),
		"email":      "john.doe@example.com",
		"first_name": "John",
		"role":       role,
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return readmodel.EventPayload{Name: "user.registered", Data: data}
}

func TestUpsertCustomer_Handle_Customer(t *testing.T) {
	customerRepo := &testutil.MockCustomerRepository{}
	h := appreadmodel.NewUpsertCustomer(customerRepo)

	err := h.Handle(newUserRegisteredPayload(t, "customer"))

	require.NoError(t, err)
	require.Len(t, customerRepo.Upserted, 1)
	assert.Equal(t, "john.doe@example.com", customerRepo.Upserted[0].Email)
}

func TestUpsertCustomer_Handle_Owner(t *testing.T) {
	customerRepo := &testutil.MockCustomerRepository{}
	h := appreadmodel.NewUpsertCustomer(customerRepo)

	err := h.Handle(newUserRegisteredPayload(t, "owner"))

	require.NoError(t, err)
	require.Len(t, customerRepo.Upserted, 1, "an owner must still be stored — owners can place orders too")
}
