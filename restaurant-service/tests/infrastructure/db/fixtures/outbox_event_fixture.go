package fixtures

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/outbox"
	"restaurant-service/tests/testutil"
)

func LoadOutboxEventFixtures(t *testing.T, db *gorm.DB) error {
	for range 5 {
		restaurantID, payload, err := restaurantUpdatedPayload()
		require.NoError(t, err)

		event := outbox.NewOutboxEvent(
			restaurantID,
			"restaurant.updated",
			payload,
		)

		err = db.Create(&event).Error
		require.NoError(t, err)
	}

	return nil
}

func restaurantUpdatedPayload() (restaurantID uuid.UUID, payload []byte, err error) {
	restaurantID = testutil.MustNewID()

	payloadMap := map[string]any{
		"restaurant_id": restaurantID,
		"event_name":    "restaurant.updated",
		"occurred_at":   time.Now().UTC(),
	}

	payload, err = json.Marshal(payloadMap)
	if err != nil {
		return restaurantID, nil, err
	}

	return restaurantID, payload, nil
}
