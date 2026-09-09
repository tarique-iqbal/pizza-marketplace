package fixtures

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"payment-service/internal/domain/outbox"
	"payment-service/tests/testutil"
)

func LoadOutboxEventFixtures(t *testing.T, db *gorm.DB) error {
	for range 5 {
		paymentID, payload, err := paymentSucceededPayload()
		require.NoError(t, err)

		event := outbox.NewOutboxEvent(
			paymentID,
			"payment.succeeded",
			payload,
		)

		err = db.Create(&event).Error
		require.NoError(t, err)
	}

	return nil
}

func paymentSucceededPayload() (paymentID uuid.UUID, payload []byte, err error) {
	paymentID = testutil.MustNewID()

	payloadMap := map[string]any{
		"payment_id":  paymentID,
		"event_name":  "payment.succeeded",
		"occurred_at": time.Now().UTC(),
	}

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return paymentID, nil, err
	}

	return paymentID, payload, nil
}
