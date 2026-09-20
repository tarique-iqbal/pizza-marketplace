package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	addressapp "customer-service/internal/application/address"
	"customer-service/internal/domain/customer"
	"customer-service/tests/infrastructure/db/fixtures"
	"customer-service/tests/testutil"
)

func setupAddresses(t *testing.T) (customer.Customer, []customer.Address) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer)

	owner := fixtures.LoadCustomerFixtures(t, db.DB)[0]
	addresses := fixtures.LoadAddressFixtures(t, db.DB, owner.ID)

	return owner, addresses
}

func TestAddressHandler_List_DefaultFirst(t *testing.T) {
	owner, addresses := setupAddresses(t)
	router := newRouter(t)

	recorder := doRequest(router, http.MethodGet, "/customers/me/addresses", &owner.ID, nil)

	require.Equal(t, http.StatusOK, recorder.Code)

	var res []addressapp.AddressResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &res))
	require.Len(t, res, 2)
	assert.True(t, res[0].IsDefault)
	assert.Equal(t, addresses[0].ID, res[0].ID)
}

func TestAddressHandler_List_OtherCustomerSeesNone(t *testing.T) {
	setupAddresses(t)
	router := newRouter(t)
	other := uuid.New()

	recorder := doRequest(router, http.MethodGet, "/customers/me/addresses", &other, nil)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `[]`, recorder.Body.String())
}

func TestAddressHandler_List_Unauthenticated(t *testing.T) {
	router := newRouter(t)

	recorder := doRequest(router, http.MethodGet, "/customers/me/addresses", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestAddressHandler_Delete_Success(t *testing.T) {
	owner, addresses := setupAddresses(t)
	router := newRouter(t)

	path := "/customers/me/addresses/" + addresses[1].ID.String()
	recorder := doRequest(router, http.MethodDelete, path, &owner.ID, nil)

	assert.Equal(t, http.StatusNoContent, recorder.Code)

	again := doRequest(router, http.MethodDelete, path, &owner.ID, nil)
	assert.Equal(t, http.StatusNotFound, again.Code)
}

func TestAddressHandler_Delete_NotOwned_ReturnsNotFound(t *testing.T) {
	_, addresses := setupAddresses(t)
	router := newRouter(t)
	other := uuid.New()

	path := "/customers/me/addresses/" + addresses[1].ID.String()
	recorder := doRequest(router, http.MethodDelete, path, &other, nil)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAddressHandler_Delete_InvalidID(t *testing.T) {
	owner, _ := setupAddresses(t)
	router := newRouter(t)

	recorder := doRequest(router, http.MethodDelete, "/customers/me/addresses/not-a-uuid", &owner.ID, nil)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAddressHandler_SetDefault_Success(t *testing.T) {
	owner, addresses := setupAddresses(t)
	router := newRouter(t)

	path := "/customers/me/addresses/" + addresses[1].ID.String() + "/default"
	recorder := doRequest(router, http.MethodPost, path, &owner.ID, nil)

	require.Equal(t, http.StatusOK, recorder.Code)

	var res addressapp.AddressResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &res))
	assert.Equal(t, addresses[1].ID, res.ID)
	assert.True(t, res.IsDefault)

	list := doRequest(router, http.MethodGet, "/customers/me/addresses", &owner.ID, nil)
	var listed []addressapp.AddressResponse
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &listed))
	require.Len(t, listed, 2)
	assert.Equal(t, addresses[1].ID, listed[0].ID)
	assert.False(t, listed[1].IsDefault)
}

func TestAddressHandler_SetDefault_NotOwned_ReturnsNotFound(t *testing.T) {
	_, addresses := setupAddresses(t)
	router := newRouter(t)
	other := uuid.New()

	path := "/customers/me/addresses/" + addresses[1].ID.String() + "/default"
	recorder := doRequest(router, http.MethodPost, path, &other, nil)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestAddressHandler_SetDefault_Unauthenticated(t *testing.T) {
	_, addresses := setupAddresses(t)
	router := newRouter(t)

	path := "/customers/me/addresses/" + addresses[1].ID.String() + "/default"
	recorder := doRequest(router, http.MethodPost, path, nil, nil)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
