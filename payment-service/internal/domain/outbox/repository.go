package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutboxRepository interface {
	WithTx(tx *gorm.DB) OutboxRepository
	Create(ctx context.Context, outbox *OutboxEvent) error
	FetchAndMarkProcessing(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
	ReleaseForRetry(ctx context.Context, id uuid.UUID, errMsg string, delay time.Duration) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
}
