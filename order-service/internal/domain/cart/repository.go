package cart

import (
	"context"

	"github.com/google/uuid"
)

type CartRepository interface {
	FindByCustomer(ctx context.Context, customerID uuid.UUID) (*Cart, error)
	Create(ctx context.Context, cart *Cart) error
	AddOrMergeItem(ctx context.Context, cartID uuid.UUID, item CartItem) error
	UpdateItemQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int16) error
	RemoveItem(ctx context.Context, cartID, itemID uuid.UUID) error
	Clear(ctx context.Context, cartID uuid.UUID) error
}
