package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/order"
	"order-service/internal/domain/readmodel"

	"pizza-marketplace/shared/logobs"
)

type PaymentFailedHandler struct {
	db        *gorm.DB
	orderRepo order.OrderRepository
}

func NewPaymentFailedHandler(db *gorm.DB, orderRepo order.OrderRepository) *PaymentFailedHandler {
	return &PaymentFailedHandler{db: db, orderRepo: orderRepo}
}

func (h *PaymentFailedHandler) Handle(ctx context.Context, evt readmodel.EventPayload) error {
	var payload paymentEventPayload
	if err := json.Unmarshal(evt.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", evt.Name, err)
	}
	if payload.SubjectType != "order" {
		return nil
	}

	cancelled := false
	err := h.db.Transaction(func(tx *gorm.DB) error {
		orderRepo := h.orderRepo.WithTx(tx)

		ord, err := orderRepo.FindByID(ctx, payload.SubjectID)
		if err != nil {
			return fmt.Errorf("failed to find order: %w", err)
		}

		if err := ord.Cancel(); err != nil {
			if errors.Is(err, order.ErrInvalidStatusTransition) {
				logobs.FromContext(ctx).WarnContext(ctx,
					"payment.failed no-op, order already cancelled or terminal",
					"order_id", ord.ID,
				)
				return nil
			}

			return fmt.Errorf("failed to cancel order: %w", err)
		}

		if err := orderRepo.Update(ctx, ord); err != nil {
			return err
		}

		cancelled = true
		return nil
	})
	if err != nil {
		return err
	}

	if cancelled {
		orderapp.RecordCancelled(ctx, orderapp.CancelSourcePaymentFailed)
	}

	return nil
}
