package fixtures

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"order-service/internal/domain/outbox"
	"order-service/tests/testutil"
)

func LoadOutboxEventFixtures(t *testing.T, db *gorm.DB) error {
	for range 5 {
		orderID, payload, err := orderConfirmedPayload()
		require.NoError(t, err)

		event := outbox.NewOutboxEvent(
			orderID,
			"order.confirmed",
			payload,
		)

		err = db.Create(&event).Error
		require.NoError(t, err)
	}

	return nil
}

func orderConfirmedPayload() (orderID uuid.UUID, payload []byte, err error) {
	orderID = testutil.MustNewID()

	payloadMap := map[string]any{
		"order_id":    orderID,
		"event_name":  "order.confirmed",
		"occurred_at": time.Now().UTC(),
	}

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return orderID, nil, err
	}

	return orderID, payload, nil
}
