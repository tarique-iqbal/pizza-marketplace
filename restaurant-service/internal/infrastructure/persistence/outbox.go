package persistence

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"restaurant-service/internal/domain/outbox"
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) outbox.OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) WithTx(tx *gorm.DB) outbox.OutboxRepository {
	return &OutboxRepository{db: tx}
}

func (r *OutboxRepository) Create(ctx context.Context, e *outbox.OutboxEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *OutboxRepository) FetchAndMarkProcessing(ctx context.Context, limit int) ([]outbox.OutboxEvent, error) {
	var events []outbox.OutboxEvent

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		table := (outbox.OutboxEvent{}).TableName()

		query := fmt.Sprintf(`
			UPDATE %s AS oe
			SET status = 'processing',
				locked_until = NOW() + INTERVAL '30 seconds',
				attempts = oe.attempts + 1
			FROM (
				SELECT id
				FROM %s
				WHERE status = 'pending'
				AND (locked_until IS NULL OR locked_until < NOW())
				AND (next_attempt_at IS NULL OR next_attempt_at <= NOW())
				ORDER BY next_attempt_at NULLS FIRST, created_at
				LIMIT ?
				FOR UPDATE SKIP LOCKED
			) AS sub
			WHERE oe.id = sub.id
			RETURNING oe.*;
		`, table, table)

		return tx.Raw(query, limit).Scan(&events).Error
	})

	return events, err
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, id int64) error {
	now := time.Now().UTC()

	res := r.db.WithContext(ctx).
		Model(&outbox.OutboxEvent{}).
		Where("id = ? AND status = 'processing'", id).
		Updates(map[string]interface{}{
			"status":          "processed",
			"processed_at":    &now,
			"locked_until":    nil,
			"next_attempt_at": nil,
		})

	return res.Error
}

func (r *OutboxRepository) ReleaseForRetry(
	ctx context.Context,
	id int64,
	errMsg string,
	delay time.Duration,
) error {
	now := time.Now().UTC()
	next := now.Add(delay)

	res := r.db.WithContext(ctx).
		Model(&outbox.OutboxEvent{}).
		Where("id = ? AND status = 'processing'", id).
		Updates(map[string]interface{}{
			"status":          "pending",
			"last_error":      errMsg,
			"locked_until":    nil,
			"next_attempt_at": &next,
		})

	return res.Error
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&outbox.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "failed",
			"last_error":   errMsg,
			"locked_until": nil,
			"processed_at": time.Now().UTC(),
		}).Error
}
