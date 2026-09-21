package commands_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customerapp "customer-service/internal/application/customer"
	"customer-service/internal/application/customer/commands"
	"customer-service/internal/domain/outbox"
	"customer-service/internal/infrastructure/persistence"
	apperr "customer-service/internal/shared/errors"
	"customer-service/tests/infrastructure/db/fixtures"
	"customer-service/tests/testutil"
)

func countOutboxEvents(t *testing.T, db *testutil.TestDB) int64 {
	t.Helper()

	var count int64
	require.NoError(t, db.DB.Model(&outbox.OutboxEvent{}).Count(&count).Error)
	return count
}

func TestUpdatePhone_Execute_SetsPhoneAndPublishes(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer, testutil.TableOutboxEvent)

	fixtureCustomers := fixtures.LoadCustomerFixtures(t, db.DB)
	target := fixtureCustomers[0]

	customerRepo := persistence.NewCustomerRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	cmd := commands.NewUpdatePhone(db.DB, customerRepo, outboxRepo)

	res, err := cmd.Execute(context.Background(), target.ID, customerapp.UpdatePhoneRequest{
		Phone: "+49 30 1234567",
	})
	require.NoError(t, err)
	require.NotNil(t, res.Phone)
	assert.Equal(t, "+49 30 1234567", *res.Phone)

	found, err := customerRepo.FindByID(context.Background(), target.ID)
	require.NoError(t, err)
	require.NotNil(t, found.Phone)
	assert.Equal(t, "+49 30 1234567", *found.Phone)

	assert.Equal(t, int64(1), countOutboxEvents(t, db))

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)
	assert.Equal(t, "customer.phone_updated", event.EventName)
	assert.Equal(t, target.ID, event.AggregateID)

	var decoded customerapp.PhoneUpdatedPayload
	require.NoError(t, json.Unmarshal(event.Payload, &decoded))
	assert.Equal(t, target.ID, decoded.CustomerID)
	assert.Equal(t, "+49 30 1234567", decoded.Phone)
}

func TestUpdatePhone_Execute_NotFound(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableCustomerAddress, testutil.TableCustomer, testutil.TableOutboxEvent)

	customerRepo := persistence.NewCustomerRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	cmd := commands.NewUpdatePhone(db.DB, customerRepo, outboxRepo)

	_, err := cmd.Execute(context.Background(), uuid.New(), customerapp.UpdatePhoneRequest{
		Phone: "+49 30 1234567",
	})

	assert.ErrorIs(t, err, apperr.ErrNotFound)
	assert.Equal(t, int64(0), countOutboxEvents(t, db))
}
