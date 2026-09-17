package fixtures

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"customer-service/internal/domain/outbox"
	"customer-service/tests/testutil"
)

func LoadOutboxEventFixtures(t *testing.T, db *gorm.DB) error {
	for range 5 {
		customerID, payload, err := phoneUpdatedPayload()
		require.NoError(t, err)

		event := outbox.NewOutboxEvent(customerID, "customer.phone_updated", payload)

		err = db.Create(&event).Error
		require.NoError(t, err)
	}

	return nil
}

func phoneUpdatedPayload() (customerID uuid.UUID, payload []byte, err error) {
	customerID = testutil.MustNewID()

	payloadMap := map[string]any{
		"customer_id": customerID,
		"phone":       "+49 30 1234567",
		"event_name":  "customer.phone_updated",
	}

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return customerID, nil, err
	}

	return customerID, payload, nil
}
