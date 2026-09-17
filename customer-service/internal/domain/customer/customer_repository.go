package customer

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerRepository interface {
	WithTx(tx *gorm.DB) CustomerRepository
	Upsert(ctx context.Context, c Customer) error
	Update(ctx context.Context, c *Customer) error
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)
}
