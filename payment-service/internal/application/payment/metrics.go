package payment

import (
	"context"
	"strings"

	"pizza-marketplace/shared/telemetry"
)

const meterScope = "payment-service/internal/application/payment"

var (
	paymentsSucceeded = telemetry.NewCounter(meterScope, "payments.succeeded", "Payments confirmed.")
	paymentsFailed    = telemetry.NewCounter(meterScope, "payments.failed", "Payments failed, by reason.")
)

func RecordSucceeded(ctx context.Context) {
	paymentsSucceeded.Add(ctx)
}

func RecordFailed(ctx context.Context, reason string) {
	paymentsFailed.Add(ctx, telemetry.Label{Key: "reason", Value: failureReasonLabel(reason)})
}

func failureReasonLabel(reason string) string {
	label := strings.TrimPrefix(reason, "payment ")
	switch label {
	case "canceled", "expired", "failed":
		return label
	default:
		return "other"
	}
}
