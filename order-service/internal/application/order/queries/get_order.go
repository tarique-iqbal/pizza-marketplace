package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
)

type GetOrder struct {
	orderRepo order.OrderRepository
}

func NewGetOrder(orderRepo order.OrderRepository) *GetOrder {
	return &GetOrder{orderRepo: orderRepo}
}

func (uc *GetOrder) Execute(
	ctx context.Context,
	orderID uuid.UUID,
	userID uuid.UUID,
	role string,
) (orderapp.OrderResponse, error) {
	var (
		ord *order.Order
		err error
	)

	if role == "owner" {
		ord, err = uc.orderRepo.FindByIDAndRestaurantOwner(ctx, orderID, userID)
	} else {
		ord, err = uc.orderRepo.FindByIDAndCustomer(ctx, orderID, userID)
	}
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return orderapp.OrderResponse{}, apperr.ErrForbidden
		}

		return orderapp.OrderResponse{}, fmt.Errorf("failed to find order: %w", err)
	}

	return orderapp.ToOrderResponse(ord), nil
}
