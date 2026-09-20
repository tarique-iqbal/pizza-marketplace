package persistence_test

import (
	"context"
	"encoding/json"
	"identity-service/internal/domain/outbox"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"owner_id":      testutil.MustNewID(),
		"business_name": "Domino's Pizza",
		"vat_number":    "DE123456121",
	}
	payload, _ := json.Marshal(payloadMap)

	event := outbox.NewOutboxEvent(
		restaurantID,
		outbox.EventRestaurantInitiated,
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
			outbox.EventRestaurantInitiated,
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

func TestOutboxRepository_FetchAndMarkProcessing_ReclaimsExpiredLock(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)

	require.NoError(t, db.DB.Model(&event).Updates(map[string]interface{}{
		"status":       outbox.StatusProcessing,
		"attempts":     1,
		"locked_until": time.Now().UTC().Add(-time.Minute),
	}).Error)

	events, err := repo.FetchAndMarkProcessing(context.Background(), 10)
	require.NoError(t, err)

	var reclaimed *outbox.OutboxEvent
	for i := range events {
		if events[i].ID == event.ID {
			reclaimed = &events[i]
		}
	}

	require.NotNil(t, reclaimed, "expired processing row must be reclaimed")
	assert.Equal(t, outbox.StatusProcessing, reclaimed.Status)
	assert.Equal(t, 2, reclaimed.Attempts)
	require.NotNil(t, reclaimed.LockedUntil)
	assert.True(t, reclaimed.LockedUntil.After(time.Now().UTC()))
}

func TestOutboxRepository_FetchAndMarkProcessing_LeavesActiveLock(t *testing.T) {
	db := testutil.DB(t)
	repo := setupOutboxRepo(t)

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)

	require.NoError(t, db.DB.Model(&event).Updates(map[string]interface{}{
		"status":       outbox.StatusProcessing,
		"attempts":     1,
		"locked_until": time.Now().UTC().Add(time.Minute),
	}).Error)

	events, err := repo.FetchAndMarkProcessing(context.Background(), 10)
	require.NoError(t, err)

	for _, e := range events {
		assert.NotEqual(t, event.ID, e.ID, "row with a live lease must not be reclaimed")
	}

	var after outbox.OutboxEvent
	require.NoError(t, db.DB.First(&after, "id = ?", event.ID).Error)
	assert.Equal(t, outbox.StatusProcessing, after.Status)
	assert.Equal(t, 1, after.Attempts)
}
