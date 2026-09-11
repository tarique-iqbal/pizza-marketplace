package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"order-service/internal/domain/order"
	"order-service/internal/domain/readmodel"
	logobs "order-service/internal/infrastructure/observability/logger"
)

type PaymentFailedHandler struct {
	db        *gorm.DB
	orderRepo order.OrderRepository
}

func NewPaymentFailedHandler(db *gorm.DB, orderRepo order.OrderRepository) *PaymentFailedHandler {
	return &PaymentFailedHandler{db: db, orderRepo: orderRepo}
}

func (h *PaymentFailedHandler) Handle(evt readmodel.EventPayload) error {
	var payload paymentEventPayload
	if err := json.Unmarshal(evt.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal %s payload: %w", evt.Name, err)
	}
	if payload.SubjectType != "order" {
		return nil
	}

	ctx := context.Background()

	return h.db.Transaction(func(tx *gorm.DB) error {
		orderRepo := h.orderRepo.WithTx(tx)

		ord, err := orderRepo.FindByID(ctx, payload.SubjectID)
		if err != nil {
			return fmt.Errorf("failed to find order: %w", err)
		}

		if err := ord.Cancel(); err != nil {
			if errors.Is(err, order.ErrInvalidStatusTransition) {
				logobs.FromContext(ctx).Warn(
					"payment.failed no-op, order already cancelled or terminal",
					"order_id", ord.ID,
				)
				return nil
			}

			return fmt.Errorf("failed to cancel order: %w", err)
		}

		return orderRepo.Update(ctx, ord)
	})
}
