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

func newPhoneUpdatedPayload(t *testing.T, customerID string, phone string) readmodel.EventPayload {
	body := map[string]any{
		"customer_id": customerID,
		"phone":       phone,
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return readmodel.EventPayload{Name: "customer.phone_updated", Data: data}
}

func TestUpdateCustomerPhone_Handle_UpdatesPhone(t *testing.T) {
	customerRepo := &testutil.MockCustomerRepository{}
	h := appreadmodel.NewUpdateCustomerPhone(customerRepo)

	customerID := testutil.MustNewID()

	err := h.Handle(newPhoneUpdatedPayload(t, customerID.String(), "+49 89 9998877"))

	require.NoError(t, err)
	assert.Equal(t, 1, customerRepo.UpdatePhoneCalls)
	assert.Equal(t, customerID, customerRepo.UpdatedPhoneID)
	assert.Equal(t, "+49 89 9998877", customerRepo.UpdatedPhone)
}

func TestUpdateCustomerPhone_Handle_PropagatesRepositoryError(t *testing.T) {
	customerRepo := &testutil.MockCustomerRepository{UpdatePhoneErr: assert.AnError}
	h := appreadmodel.NewUpdateCustomerPhone(customerRepo)

	err := h.Handle(newPhoneUpdatedPayload(t, testutil.MustNewID().String(), "+49 89 9998877"))

	assert.ErrorIs(t, err, assert.AnError)
}
