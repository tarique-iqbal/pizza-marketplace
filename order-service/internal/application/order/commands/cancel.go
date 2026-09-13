package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/order"
	logobs "order-service/internal/infrastructure/observability/logger"
	apperr "order-service/internal/shared/errors"
)

type Cancel struct {
	orderRepo       order.OrderRepository
	paymentProvider order.PaymentProvider
}

func NewCancel(orderRepo order.OrderRepository, paymentProvider order.PaymentProvider) *Cancel {
	return &Cancel{orderRepo: orderRepo, paymentProvider: paymentProvider}
}

func (uc *Cancel) Execute(
	ctx context.Context,
	orderID, userID uuid.UUID,
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

	// The local status can be outdated, so re-check with payment-service directly.
	// Only a succeeded payment blocks cancelling, since that money is already captured.
	if ord.PaymentID != nil {
		paymentStatus, err := uc.paymentProvider.GetPaymentStatus(ctx, *ord.PaymentID)
		if err != nil {
			return orderapp.OrderResponse{}, fmt.Errorf("failed to check payment status: %w", err)
		}
		if paymentStatus == order.PaymentStatusSucceeded {
			return orderapp.OrderResponse{}, fmt.Errorf("payment already succeeded: %w", apperr.ErrConflict)
		}
	}

	if err := ord.Cancel(); err != nil {
		return orderapp.OrderResponse{}, fmt.Errorf("%w: %w", err, apperr.ErrConflict)
	}

	if err := uc.orderRepo.Update(ctx, ord); err != nil {
		return orderapp.OrderResponse{}, fmt.Errorf("failed to update order: %w", err)
	}

	// Best-effort: the order stays cancelled even if the gateway cancel call fails.
	if ord.PaymentID != nil {
		if err := uc.paymentProvider.CancelPayment(ctx, *ord.PaymentID); err != nil {
			logobs.FromContext(ctx).Warn("failed to cancel payment", "order_id", ord.ID, "error", err)
		}
	}

	return orderapp.ToOrderResponse(ord), nil
}
