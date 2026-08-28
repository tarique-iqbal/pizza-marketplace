package commands_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/outbox"
)

func firstOutboxEvent(t *testing.T, db *gorm.DB, restaurantID uuid.UUID, eventName string) outbox.OutboxEvent {
	t.Helper()

	var found outbox.OutboxEvent
	err := db.Where("aggregate_id = ? AND event_name = ?", restaurantID, eventName).First(&found).Error
	require.NoError(t, err)

	return found
}
