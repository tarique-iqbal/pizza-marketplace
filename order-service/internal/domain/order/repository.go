package order

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository interface {
	WithTx(tx *gorm.DB) OrderRepository
	Create(ctx context.Context, order *Order) error
	Update(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
	FindByIDAndCustomer(ctx context.Context, id, customerID uuid.UUID) (*Order, error)
	FindByIDAndRestaurantOwner(ctx context.Context, id, ownerID uuid.UUID) (*Order, error)
	ListByCustomer(ctx context.Context, customerID uuid.UUID, after *PageCursor, limit int) ([]Order, error)
	ListByRestaurant(
		ctx context.Context,
		restaurantID, ownerID uuid.UUID,
		after *PageCursor,
		limit int,
	) ([]Order, error)
}
