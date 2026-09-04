package readmodel

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PizzaRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Pizza, error)
	Upsert(ctx context.Context, pizza Pizza) error
	// Delete is guarded by updatedAt — a no-op if the stored row is already newer.
	Delete(ctx context.Context, id uuid.UUID, updatedAt time.Time) error
}
