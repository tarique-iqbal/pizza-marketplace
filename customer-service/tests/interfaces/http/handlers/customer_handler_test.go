package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customerapp "customer-service/internal/application/customer"
	"customer-service/internal/domain/outbox"
	"customer-service/tests/infrastructure/db/fixtures"
	"customer-service/tests/testutil"
)

func TestCustomerHandler_GetProfile_Success(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer)
	target := fixtures.LoadCustomerFixtures(t, db.DB)[1]
	router := newRouter(t)

	recorder := doRequest(router, http.MethodGet, "/customers/me", &target.ID, nil)

	require.Equal(t, http.StatusOK, recorder.Code)

	var res customerapp.GetProfileResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &res))
	assert.Equal(t, target.ID, res.ID)
	assert.Equal(t, "with-phone@example.com", res.Email)
	require.NotNil(t, res.Phone)
	assert.Equal(t, "+49 40 12345678", *res.Phone)
}

func TestCustomerHandler_GetProfile_Unauthenticated(t *testing.T) {
	router := newRouter(t)

	recorder := doRequest(router, http.MethodGet, "/customers/me", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestCustomerHandler_UpdatePhone_Success(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer, testutil.TableOutboxEvent)
	target := fixtures.LoadCustomerFixtures(t, db.DB)[0]
	router := newRouter(t)

	recorder := doRequest(
		router, http.MethodPatch, "/customers/me/phone", &target.ID, []byte(`{"phone":"+49 30 1234567"}`),
	)

	require.Equal(t, http.StatusOK, recorder.Code)

	var res customerapp.GetProfileResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &res))
	require.NotNil(t, res.Phone)
	assert.Equal(t, "+49 30 1234567", *res.Phone)

	var count int64
	require.NoError(t, db.DB.Model(&outbox.OutboxEvent{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestCustomerHandler_UpdatePhone_OldPathNotRouted(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer)
	target := fixtures.LoadCustomerFixtures(t, db.DB)[0]
	router := newRouter(t)

	recorder := doRequest(
		router, http.MethodPatch, "/customers/me", &target.ID, []byte(`{"phone":"+49 30 1234567"}`),
	)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestCustomerHandler_UpdatePhone_InvalidBody(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer)
	target := fixtures.LoadCustomerFixtures(t, db.DB)[0]
	router := newRouter(t)

	recorder := doRequest(router, http.MethodPatch, "/customers/me/phone", &target.ID, []byte(`{}`))

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
}

func TestCustomerHandler_UpdatePhone_Unauthenticated(t *testing.T) {
	router := newRouter(t)

	recorder := doRequest(
		router, http.MethodPatch, "/customers/me/phone", nil, []byte(`{"phone":"+49 30 1234567"}`),
	)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
