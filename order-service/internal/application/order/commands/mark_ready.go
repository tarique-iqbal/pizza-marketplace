package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
)

type MarkReady struct {
	orderRepo order.OrderRepository
}

func NewMarkReady(orderRepo order.OrderRepository) *MarkReady {
	return &MarkReady{orderRepo: orderRepo}
}

func (uc *MarkReady) Execute(
	ctx context.Context,
	orderID, ownerID uuid.UUID,
) (orderapp.OrderResponse, error) {
	ord, err := uc.orderRepo.FindByIDAndRestaurantOwner(ctx, orderID, ownerID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return orderapp.OrderResponse{}, apperr.ErrForbidden
		}

		return orderapp.OrderResponse{}, fmt.Errorf("failed to find order: %w", err)
	}

	if err := ord.MarkReady(); err != nil {
		return orderapp.OrderResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
	}

	if err := uc.orderRepo.Update(ctx, ord); err != nil {
		return orderapp.OrderResponse{}, fmt.Errorf("failed to update order: %w", err)
	}

	return orderapp.ToOrderResponse(ord), nil
}
