package customer

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddressRepository interface {
	WithTx(tx *gorm.DB) AddressRepository
	Create(ctx context.Context, a *Address) error
	Delete(ctx context.Context, id, customerID uuid.UUID) error
	FindByID(ctx context.Context, id, customerID uuid.UUID) (*Address, error)
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]Address, error)
	UnsetDefault(ctx context.Context, customerID uuid.UUID) error
	SetDefault(ctx context.Context, id uuid.UUID) error
	ExistsForCustomer(
		ctx context.Context,
		customerID uuid.UUID,
		house, street, city, postalCode string,
	) (bool, error)
}
