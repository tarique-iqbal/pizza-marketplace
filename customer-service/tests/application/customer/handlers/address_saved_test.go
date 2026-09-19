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

func newAddressSavedPayload(t *testing.T) customer.EventPayload {
	body := map[string]any{
		"customer_id": testutil.MustNewID().String(),
		"house":       "10",
		"street":      "Main St",
		"city":        "Berlin",
		"postal_code": "10115",
	}

	data, err := json.Marshal(body)
	require.NoError(t, err)

	return customer.EventPayload{Name: "order.address_saved", Data: data}
}

func TestAddressSaved_Handle_NoExistingAddresses_CreatesAsDefault(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{}
	h := apphandlers.NewAddressSaved(addressRepo)

	err := h.Handle(newAddressSavedPayload(t))

	require.NoError(t, err)
	require.Len(t, addressRepo.Created, 1)
	assert.Equal(t, "Main St", addressRepo.Created[0].Street)
	assert.True(t, addressRepo.Created[0].IsDefault)
}

func TestAddressSaved_Handle_ExistingAddresses_CreatesAsNonDefault(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{
		ListByCustomerResult: []customer.Address{{Street: "Other St"}},
	}
	h := apphandlers.NewAddressSaved(addressRepo)

	err := h.Handle(newAddressSavedPayload(t))

	require.NoError(t, err)
	require.Len(t, addressRepo.Created, 1)
	assert.False(t, addressRepo.Created[0].IsDefault)
}

func TestAddressSaved_Handle_AlreadyExists_NoOp(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{ExistsResult: true}
	h := apphandlers.NewAddressSaved(addressRepo)

	err := h.Handle(newAddressSavedPayload(t))

	require.NoError(t, err)
	assert.Empty(t, addressRepo.Created)
}

func TestAddressSaved_Handle_PropagatesExistsError(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{ExistsErr: assert.AnError}
	h := apphandlers.NewAddressSaved(addressRepo)

	err := h.Handle(newAddressSavedPayload(t))

	assert.ErrorIs(t, err, assert.AnError)
}

func TestAddressSaved_Handle_PropagatesListError(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{ListByCustomerErr: assert.AnError}
	h := apphandlers.NewAddressSaved(addressRepo)

	err := h.Handle(newAddressSavedPayload(t))

	assert.ErrorIs(t, err, assert.AnError)
}

func TestAddressSaved_Handle_PropagatesCreateError(t *testing.T) {
	addressRepo := &testutil.MockAddressRepository{CreateErr: assert.AnError}
	h := apphandlers.NewAddressSaved(addressRepo)

	err := h.Handle(newAddressSavedPayload(t))

	assert.ErrorIs(t, err, assert.AnError)
}
