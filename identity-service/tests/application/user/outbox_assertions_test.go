package user_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"identity-service/internal/domain/outbox"
)

func firstOutboxEvent(t *testing.T, db *gorm.DB, aggregateID uuid.UUID, eventName string) outbox.OutboxEvent {
	t.Helper()

	var found outbox.OutboxEvent
	err := db.Where("aggregate_id = ? AND event_name = ?", aggregateID, eventName).First(&found).Error
	require.NoError(t, err)

	return found
}
