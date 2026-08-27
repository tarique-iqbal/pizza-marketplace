package persistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/tests/infrastructure/db/fixtures"
	"restaurant-service/tests/testutil"
)

func setupOutboxRepo(t *testing.T) outbox.OutboxRepository {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableOutboxEvent)

	_ = fixtures.LoadOutboxEventFixtures(t, db.DB)

	return persistence.NewOutboxRepository(db.DB)
}

func TestOutboxRepository_Create(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	restaurantID := testutil.MustNewID()
	payloadMap := map[string]interface{}{
		"restaurant_id": restaurantID,
		"event_name":    "restaurant.launched",
	}
	payload, _ := json.Marshal(payloadMap)

	event := outbox.NewOutboxEvent(
		restaurantID,
		"restaurant.launched",
		payload,
	)

	err := repo.Create(context.Background(), &event)

	require.NoError(t, err)
	assert.NotZero(t, event.ID)

	var found outbox.OutboxEvent
	err = db.DB.First(&found, event.ID).Error
	require.NoError(t, err)

	assert.Equal(t, outbox.StatusPending, found.Status)
	assert.NotZero(t, found.CreatedAt)
}

func TestOutboxRepository_FetchAndMarkProcessing(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	events, err := repo.FetchAndMarkProcessing(context.Background(), 1)

	require.NoError(t, err)
	require.Len(t, events, 1)

	e := events[0]

	// runtime checks
	assert.Equal(t, outbox.StatusProcessing, e.Status)
	assert.NotNil(t, e.LockedUntil)
	assert.Equal(t, 1, e.Attempts)

	// DB verification (critical)
	var dbEvent outbox.OutboxEvent
	err = db.DB.First(&dbEvent, e.ID).Error
	require.NoError(t, err)

	assert.Equal(t, outbox.StatusProcessing, dbEvent.Status)
	assert.NotNil(t, dbEvent.LockedUntil)
	assert.Equal(t, 1, dbEvent.Attempts)
}

func TestOutboxRepository_FetchAndMarkProcessing_Limit(t *testing.T) {
	repo := setupOutboxRepo(t)

	// insert extra events
	for range 3 {
		e := outbox.NewOutboxEvent(
			testutil.MustNewID(),
			"restaurant.updated",
			[]byte(`{}`),
		)
		require.NoError(t, repo.Create(context.Background(), &e))
	}

	events, err := repo.FetchAndMarkProcessing(context.Background(), 2)

	require.NoError(t, err)
	assert.Len(t, events, 2)
}

func TestOutboxRepository_MarkProcessed(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)

	event.Status = outbox.StatusProcessing
	require.NoError(t, db.DB.Save(&event).Error)

	err := repo.MarkProcessed(context.Background(), event.ID)
	require.NoError(t, err)

	var updated outbox.OutboxEvent
	require.NoError(t, db.DB.First(&updated, event.ID).Error)

	assert.Equal(t, outbox.StatusProcessed, updated.Status)
	assert.NotNil(t, updated.ProcessedAt)
	assert.Nil(t, updated.LockedUntil)
}

func TestOutboxRepository_ReleaseForRetry(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)

	// simulate processing state
	require.NoError(t, db.DB.Model(&event).Updates(map[string]interface{}{
		"status":       outbox.StatusProcessing,
		"locked_until": time.Now().UTC().Add(30 * time.Second),
	}).Error)

	err := repo.ReleaseForRetry(context.Background(), event.ID, "temporary failure", 10*time.Second)
	require.NoError(t, err)

	var updated outbox.OutboxEvent
	require.NoError(t, db.DB.First(&updated, event.ID).Error)

	require.NotNil(t, updated.LastError)
	assert.Equal(t, outbox.StatusPending, updated.Status)
	assert.Equal(t, "temporary failure", *updated.LastError)
	assert.Nil(t, updated.LockedUntil)
}

func TestOutboxRepository_MarkFailed(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)

	err := repo.MarkFailed(context.Background(), event.ID, "permanent failure")
	require.NoError(t, err)

	var updated outbox.OutboxEvent
	require.NoError(t, db.DB.First(&updated, event.ID).Error)

	require.NotNil(t, updated.LastError)
	assert.Equal(t, outbox.StatusFailed, updated.Status)
	assert.Equal(t, "permanent failure", *updated.LastError)
	assert.Nil(t, updated.LockedUntil)
	assert.NotNil(t, updated.ProcessedAt)
}

func TestOutboxRepository_FetchAndMarkProcessing_SkipLocked(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)

	// lock one event manually
	require.NoError(t, db.DB.Model(&event).Updates(map[string]interface{}{
		"status":       outbox.StatusProcessing,
		"locked_until": time.Now().UTC().Add(30 * time.Second),
	}).Error)

	events, err := repo.FetchAndMarkProcessing(context.Background(), 10)

	require.NoError(t, err)

	// locked event should NOT be returned
	for _, e := range events {
		assert.NotEqual(t, event.ID, e.ID)
	}
}

func TestOutboxRepository_WithTx_CommitsWithOuterTransaction(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableOutboxEvent)

	repo := persistence.NewOutboxRepository(db.DB)
	restaurantID := testutil.MustNewID()

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		event := outbox.NewOutboxEvent(restaurantID, "restaurant.updated", []byte(`{}`))
		return repo.WithTx(tx).Create(context.Background(), &event)
	})
	require.NoError(t, err)

	var found outbox.OutboxEvent
	err = db.DB.Where("aggregate_id = ?", restaurantID).First(&found).Error
	require.NoError(t, err)
	assert.Equal(t, outbox.StatusPending, found.Status)
}
