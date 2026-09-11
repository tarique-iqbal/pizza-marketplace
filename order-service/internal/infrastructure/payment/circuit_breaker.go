package payment

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"

	"order-service/internal/domain/order"
)

type CircuitBreakerProvider struct {
	inner         order.PaymentProvider
	createBreaker *gobreaker.CircuitBreaker[order.CreatePaymentResult]
	cancelBreaker *gobreaker.CircuitBreaker[struct{}]
}

func NewCircuitBreakerProvider(inner order.PaymentProvider) *CircuitBreakerProvider {
	readyToTrip := func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= 5
	}

	return &CircuitBreakerProvider{
		inner: inner,
		createBreaker: gobreaker.NewCircuitBreaker[order.CreatePaymentResult](gobreaker.Settings{
			Name:        "payment-service.create_payment",
			MaxRequests: 1,
			Interval:    60 * time.Second,
			Timeout:     30 * time.Second,
			ReadyToTrip: readyToTrip,
		}),
		cancelBreaker: gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{
			Name:        "payment-service.cancel_payment",
			MaxRequests: 1,
			Interval:    60 * time.Second,
			Timeout:     30 * time.Second,
			ReadyToTrip: readyToTrip,
		}),
	}
}

func (p *CircuitBreakerProvider) CreatePayment(
	ctx context.Context,
	req order.CreatePaymentRequest,
) (order.CreatePaymentResult, error) {
	result, err := p.createBreaker.Execute(func() (order.CreatePaymentResult, error) {
		return p.inner.CreatePayment(ctx, req)
	})
	if isBreakerOpen(err) {
		return order.CreatePaymentResult{}, order.ErrPaymentServiceUnavailable
	}

	return result, err
}

func (p *CircuitBreakerProvider) CancelPayment(ctx context.Context, paymentID string) error {
	_, err := p.cancelBreaker.Execute(func() (struct{}, error) {
		return struct{}{}, p.inner.CancelPayment(ctx, paymentID)
	})
	if isBreakerOpen(err) {
		return order.ErrPaymentServiceUnavailable
	}

	return err
}

func isBreakerOpen(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests)
}
