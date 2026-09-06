package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	cartapp "order-service/internal/application/cart"
	"order-service/internal/domain/cart"
	apperr "order-service/internal/shared/errors"
)

type UpdateItemQuantity struct {
	cartRepo cart.CartRepository
}

func NewUpdateItemQuantity(cartRepo cart.CartRepository) *UpdateItemQuantity {
	return &UpdateItemQuantity{cartRepo: cartRepo}
}

func (uc *UpdateItemQuantity) Execute(
	ctx context.Context,
	customerID, itemID uuid.UUID,
	input cartapp.UpdateItemQuantityRequest,
) (cartapp.UpdateItemQuantityResponse, error) {
	c, err := uc.cartRepo.FindByCustomer(ctx, customerID)
	if err != nil {
		return cartapp.UpdateItemQuantityResponse{}, fmt.Errorf("failed to look up cart: %w", err)
	}
	if c == nil {
		return cartapp.UpdateItemQuantityResponse{}, fmt.Errorf("cart not found: %w", apperr.ErrNotFound)
	}

	if err := uc.cartRepo.UpdateItemQuantity(ctx, c.ID, itemID, input.Quantity); err != nil {
		return cartapp.UpdateItemQuantityResponse{}, fmt.Errorf("failed to update cart item quantity: %w", err)
	}

	return cartapp.UpdateItemQuantityResponse{ItemID: itemID, Quantity: input.Quantity}, nil
}
