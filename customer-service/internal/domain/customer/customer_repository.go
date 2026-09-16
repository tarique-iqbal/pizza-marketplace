package customer

import (
	"context"

	"github.com/google/uuid"
)

type CustomerRepository interface {
	Upsert(ctx context.Context, c Customer) error
	Update(ctx context.Context, c *Customer) error
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)
}
