package payment_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-service/internal/domain/payment"
)

func TestNewPayment_SetsDefaults(t *testing.T) {
	id := uuid.New()
	subjectID := uuid.New()
	restaurantID := uuid.New()
	customerID := uuid.New()
	amount := decimal.NewFromFloat(24.50)
	platformFee := decimal.NewFromFloat(1.50)

	p := payment.NewPayment(
		id, subjectID, restaurantID, customerID,
		"order",
		amount,
		"EUR",
		platformFee,
		"mollie",
	)

	assert.Equal(t, id, p.ID)
	assert.Equal(t, "order", p.SubjectType)
	assert.Equal(t, subjectID, p.SubjectID)
	assert.Equal(t, restaurantID, p.RestaurantID)
	assert.Equal(t, customerID, p.CustomerID)
	assert.True(t, amount.Equal(p.Amount))
	assert.Equal(t, "EUR", p.Currency)
	assert.True(t, platformFee.Equal(p.PlatformFee))
	assert.Equal(t, payment.StatusPending, p.Status)
	assert.Equal(t, "mollie", p.Gateway)
	assert.Nil(t, p.GatewayPaymentID)
	assert.Nil(t, p.FailureReason)
	assert.Nil(t, p.UpdatedAt)
}

func TestPayment_AttachGatewayReference_SetsField(t *testing.T) {
	p := &payment.Payment{Status: payment.StatusPending}

	p.AttachGatewayReference("tr_abc123")

	require.NotNil(t, p.GatewayPaymentID)
	assert.Equal(t, "tr_abc123", *p.GatewayPaymentID)
}

func TestPayment_MarkSucceeded_TransitionsFromPending(t *testing.T) {
	p := &payment.Payment{Status: payment.StatusPending}

	err := p.MarkSucceeded()

	require.NoError(t, err)
	assert.Equal(t, payment.StatusSucceeded, p.Status)
}

func TestPayment_MarkSucceeded_FailsIfNotPending(t *testing.T) {
	for _, status := range []payment.PaymentStatus{
		payment.StatusSucceeded,
		payment.StatusFailed,
	} {
		p := &payment.Payment{Status: status}

		err := p.MarkSucceeded()

		require.ErrorIs(t, err, payment.ErrInvalidStatusTransition)
		assert.Equal(t, status, p.Status)
	}
}

func TestPayment_MarkFailed_TransitionsFromPending(t *testing.T) {
	p := &payment.Payment{Status: payment.StatusPending}

	err := p.MarkFailed("card declined")

	require.NoError(t, err)
	assert.Equal(t, payment.StatusFailed, p.Status)
	require.NotNil(t, p.FailureReason)
	assert.Equal(t, "card declined", *p.FailureReason)
}

func TestPayment_MarkFailed_FailsIfNotPending(t *testing.T) {
	for _, status := range []payment.PaymentStatus{
		payment.StatusSucceeded,
		payment.StatusFailed,
	} {
		p := &payment.Payment{Status: status}

		err := p.MarkFailed("card declined")

		require.ErrorIs(t, err, payment.ErrInvalidStatusTransition)
		assert.Equal(t, status, p.Status)
		assert.Nil(t, p.FailureReason)
	}
}
