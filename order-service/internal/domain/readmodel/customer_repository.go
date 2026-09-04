package readmodel

import (
	"context"

	"github.com/google/uuid"
)

type CustomerRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	Upsert(ctx context.Context, customer Customer) error
}
