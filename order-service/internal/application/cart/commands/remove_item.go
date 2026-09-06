package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"order-service/internal/domain/cart"
	apperr "order-service/internal/shared/errors"
)

type RemoveItem struct {
	cartRepo cart.CartRepository
}

func NewRemoveItem(cartRepo cart.CartRepository) *RemoveItem {
	return &RemoveItem{cartRepo: cartRepo}
}

func (uc *RemoveItem) Execute(ctx context.Context, customerID, itemID uuid.UUID) error {
	c, err := uc.cartRepo.FindByCustomer(ctx, customerID)
	if err != nil {
		return fmt.Errorf("failed to look up cart: %w", err)
	}
	if c == nil {
		return fmt.Errorf("cart not found: %w", apperr.ErrNotFound)
	}

	if err := uc.cartRepo.RemoveItem(ctx, c.ID, itemID); err != nil {
		return fmt.Errorf("failed to remove cart item: %w", err)
	}

	return nil
}
