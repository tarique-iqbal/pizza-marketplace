package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	orderapp "order-service/internal/application/order"
	"order-service/internal/domain/order"
	"order-service/internal/domain/outbox"
	"order-service/internal/domain/readmodel"
	logobs "order-service/internal/infrastructure/observability/logger"
	"order-service/internal/shared/event"
)

type paymentEventPayload struct {
	SubjectType string    `json:"subject_type"`
	SubjectID   uuid.UUID `json:"subject_id"`
}

type PaymentSucceededHandler struct {
	db             *gorm.DB
	orderRepo      order.OrderRepository
	outboxRepo     outbox.OutboxRepository
	restaurantRepo readmodel.RestaurantRepository
}

func NewPaymentSucceededHandler(
	db *gorm.DB,
	orderRepo order.OrderRepository,
	outboxRepo outbox.OutboxRepository,
	restaurantRepo readmodel.RestaurantRepository,
) *PaymentSucceededHandler {
	return &PaymentSucceededHandler{
		db:             db,
		orderRepo:      orderRepo,
		outboxRepo:     outboxRepo,
		restaurantRepo: restaurantRepo,
	}
}

func (h *PaymentSucceededHandler) Handle(evt readmodel.EventPayload) error {
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

		if err := ord.Confirm(); err != nil {
			if errors.Is(err, order.ErrInvalidStatusTransition) {
				logobs.FromContext(ctx).Warn(
					"payment.succeeded no-op, order already confirmed or terminal",
					"order_id", ord.ID,
				)
				return nil
			}

			return fmt.Errorf("failed to confirm order: %w", err)
		}

		if err := orderRepo.Update(ctx, ord); err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		restaurant, err := h.restaurantRepo.FindByID(ctx, ord.RestaurantID)
		if err != nil {
			return fmt.Errorf("failed to find restaurant: %w", err)
		}

		items := make([]orderapp.OrderItemPayload, 0, len(ord.Items))
		for _, item := range ord.Items {
			items = append(items, orderapp.OrderItemPayload{Name: item.PizzaName, Quantity: item.Quantity})
		}

		enricher := func(e order.DomainEvent) (event.Event, bool) {
			confirmed, ok := e.(order.OrderConfirmed)
			if !ok {
				return nil, false
			}

			return orderapp.OrderConfirmedPayload{
				OrderID:        confirmed.OrderID,
				RestaurantID:   confirmed.RestaurantID,
				RestaurantName: restaurant.Name,
				CustomerEmail:  ord.ContactEmail,
				OwnerEmail:     restaurant.OwnerEmail,
				Items:          items,
				Total:          ord.Total.StringFixed(2),
				Currency:       ord.Currency,
				ConfirmedAt:    confirmed.OccurredAt,
			}, true
		}

		return orderapp.DispatchEventsTx(ctx, h.outboxRepo.WithTx(tx), ord, enricher)
	})
}
