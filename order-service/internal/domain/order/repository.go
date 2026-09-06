package order

import (
	"context"

	"github.com/google/uuid"
)

type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	Update(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
	FindByIDAndCustomer(ctx context.Context, id, customerID uuid.UUID) (*Order, error)
	FindByIDAndRestaurantOwner(ctx context.Context, id, ownerID uuid.UUID) (*Order, error)
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]Order, error)
	ListByRestaurantOwner(ctx context.Context, ownerID uuid.UUID) ([]Order, error)
}
