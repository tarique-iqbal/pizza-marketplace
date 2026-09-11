package payment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/order"
	"order-service/internal/infrastructure/payment"
)

type fakeProvider struct {
	callCount    int
	createErr    error
	createResult order.CreatePaymentResult
	cancelErr    error
}

func (f *fakeProvider) CreatePayment(
	_ context.Context,
	_ order.CreatePaymentRequest,
) (order.CreatePaymentResult, error) {
	f.callCount++

	return f.createResult, f.createErr
}

func (f *fakeProvider) CancelPayment(_ context.Context, _ string) error {
	f.callCount++

	return f.cancelErr
}

func TestCircuitBreakerProvider_CreatePayment_PassesThroughSuccess(t *testing.T) {
	fake := &fakeProvider{createResult: order.CreatePaymentResult{PaymentID: "p1", CheckoutURL: "url"}}
	cb := payment.NewCircuitBreakerProvider(fake)

	result, err := cb.CreatePayment(context.Background(), order.CreatePaymentRequest{})
	require.NoError(t, err)
	assert.Equal(t, "p1", result.PaymentID)
	assert.Equal(t, "url", result.CheckoutURL)
	assert.Equal(t, 1, fake.callCount)
}

func TestCircuitBreakerProvider_CreatePayment_TripsAfterConsecutiveFailures(t *testing.T) {
	fake := &fakeProvider{createErr: errors.New("boom")}
	cb := payment.NewCircuitBreakerProvider(fake)

	for i := 0; i < 5; i++ {
		_, err := cb.CreatePayment(context.Background(), order.CreatePaymentRequest{})
		require.Error(t, err)
	}
	require.Equal(t, 5, fake.callCount)

	_, err := cb.CreatePayment(context.Background(), order.CreatePaymentRequest{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, order.ErrPaymentServiceUnavailable))
	assert.Equal(t, 5, fake.callCount, "breaker should short-circuit without calling the inner provider again")
}

func TestCircuitBreakerProvider_CancelPayment_PassesThroughSuccess(t *testing.T) {
	fake := &fakeProvider{}
	cb := payment.NewCircuitBreakerProvider(fake)

	err := cb.CancelPayment(context.Background(), "pay_123")
	require.NoError(t, err)
	assert.Equal(t, 1, fake.callCount)
}

func TestCircuitBreakerProvider_CancelPayment_TripsAfterConsecutiveFailures(t *testing.T) {
	fake := &fakeProvider{cancelErr: errors.New("boom")}
	cb := payment.NewCircuitBreakerProvider(fake)

	for i := 0; i < 5; i++ {
		err := cb.CancelPayment(context.Background(), "pay_123")
		require.Error(t, err)
	}
	require.Equal(t, 5, fake.callCount)

	err := cb.CancelPayment(context.Background(), "pay_123")
	require.Error(t, err)
	assert.True(t, errors.Is(err, order.ErrPaymentServiceUnavailable))
	assert.Equal(t, 5, fake.callCount, "breaker should short-circuit without calling the inner provider again")
}
