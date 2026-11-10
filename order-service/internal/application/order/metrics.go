package order

import (
	"context"

	"order-service/internal/domain/order"

	"pizza-marketplace/shared/telemetry"
)

const meterScope = "order-service/internal/application/order"

type CancelSource string

const (
	CancelSourceCustomer      CancelSource = "customer"
	CancelSourceOwner         CancelSource = "owner"
	CancelSourcePaymentFailed CancelSource = "payment_failed"
)

var (
	ordersPlaced    = telemetry.NewCounter(meterScope, "orders.placed", "Orders committed at checkout.")
	ordersConfirmed = telemetry.NewCounter(meterScope, "orders.confirmed", "Orders confirmed after payment.")
	ordersCancelled = telemetry.NewCounter(meterScope, "orders.cancelled", "Orders cancelled, by source.")
	ordersCompleted = telemetry.NewCounter(meterScope, "orders.completed", "Orders completed by the owner.")

	orderCompletionDuration = telemetry.NewHistogram(meterScope, "order.completion.duration", "s",
		"Time from placing an order to completing it.",
		5*60, 10*60, 15*60, 20*60, 30*60, 45*60, 60*60, 90*60, 120*60,
	)
)

func RecordPlaced(ctx context.Context, ord *order.Order) {
	ordersPlaced.Add(ctx, fulfillmentLabel(ord))
}

func RecordConfirmed(ctx context.Context, ord *order.Order) {
	ordersConfirmed.Add(ctx, fulfillmentLabel(ord))
}

func RecordCancelled(ctx context.Context, source CancelSource) {
	ordersCancelled.Add(ctx, telemetry.Label{Key: "source", Value: string(source)})
}

func RecordCompleted(ctx context.Context, ord *order.Order) {
	ordersCompleted.Add(ctx, fulfillmentLabel(ord))

	if ord.CompletedAt != nil && !ord.PlacedAt.IsZero() {
		duration := ord.CompletedAt.Sub(ord.PlacedAt).Seconds()
		orderCompletionDuration.Record(ctx, duration, fulfillmentLabel(ord))
	}
}

func fulfillmentLabel(ord *order.Order) telemetry.Label {
	return telemetry.Label{Key: "fulfillment", Value: string(ord.Fulfillment)}
}
