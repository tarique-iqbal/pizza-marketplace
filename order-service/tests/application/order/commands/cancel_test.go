package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/application/order/commands"
	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
	"order-service/tests/testutil"
)

type cancelPaymentFake struct {
	cancelledPaymentIDs []string
	cancelErr           error
	statusResult        order.PaymentStatus
	statusErr           error
}

func (f *cancelPaymentFake) CreatePayment(
	_ context.Context,
	_ order.CreatePaymentRequest,
) (order.CreatePaymentResult, error) {
	return order.CreatePaymentResult{}, nil
}

func (f *cancelPaymentFake) CancelPayment(_ context.Context, paymentID string) error {
	f.cancelledPaymentIDs = append(f.cancelledPaymentIDs, paymentID)
	return f.cancelErr
}

func (f *cancelPaymentFake) GetPaymentStatus(_ context.Context, _ string) (order.PaymentStatus, error) {
	if f.statusResult == "" {
		return order.PaymentStatusPending, f.statusErr
	}

	return f.statusResult, f.statusErr
}

func TestCancel_AsCustomer_Success(t *testing.T) {
	paymentID := "pay_123"
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending, PaymentID: &paymentID,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	payment := &cancelPaymentFake{}
	uc := commands.NewCancel(repo, payment)

	res, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "customer")

	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, res.Status)
	require.Len(t, repo.Updated, 1)
	assert.Equal(t, []string{"pay_123"}, payment.cancelledPaymentIDs)
}

func TestCancel_AsOwner_Success(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndRestaurantOwnerResult: ord}
	uc := commands.NewCancel(repo, &cancelPaymentFake{})

	res, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "owner")

	require.NoError(t, err)
	assert.Equal(t, order.StatusCancelled, res.Status)
}

func TestCancel_NoPaymentIDYet_SkipsPaymentCancel(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	payment := &cancelPaymentFake{}
	uc := commands.NewCancel(repo, payment)

	_, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "customer")

	require.NoError(t, err)
	assert.Empty(t, payment.cancelledPaymentIDs)
}

func TestCancel_PaymentCancelFails_OrderStillCancelled(t *testing.T) {
	paymentID := "pay_123"
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending, PaymentID: &paymentID,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	payment := &cancelPaymentFake{cancelErr: errors.New("mollie unavailable")}
	uc := commands.NewCancel(repo, payment)

	res, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "customer")

	require.NoError(t, err, "a failed payment cancel must not fail the order cancel")
	assert.Equal(t, order.StatusCancelled, res.Status)
}

func TestCancel_PaymentAlreadySucceeded_ReturnsConflict(t *testing.T) {
	paymentID := "pay_123"
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending, PaymentID: &paymentID,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	payment := &cancelPaymentFake{statusResult: order.PaymentStatusSucceeded}
	uc := commands.NewCancel(repo, payment)

	_, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "customer")

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.Empty(t, repo.Updated, "must not cancel an order whose payment already succeeded")
	assert.Empty(t, payment.cancelledPaymentIDs)
}

func TestCancel_PaymentAlreadyFailed_StillCancels(t *testing.T) {
	paymentID := "pay_123"
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending, PaymentID: &paymentID,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	payment := &cancelPaymentFake{statusResult: order.PaymentStatusFailed}
	uc := commands.NewCancel(repo, payment)

	res, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "customer")

	require.NoError(t, err, "a failed payment never captured money, so cancel should still succeed")
	assert.Equal(t, order.StatusCancelled, res.Status)
}

func TestCancel_PaymentStatusCheckFails_FailsClosed(t *testing.T) {
	paymentID := "pay_123"
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusPending, PaymentID: &paymentID,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	payment := &cancelPaymentFake{statusResult: order.PaymentStatusPending, statusErr: order.ErrPaymentServiceUnavailable}
	uc := commands.NewCancel(repo, payment)

	_, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "customer")

	require.Error(t, err)
	assert.ErrorIs(t, err, order.ErrPaymentServiceUnavailable)
	assert.Empty(t, repo.Updated, "must not cancel when the payment status can't be verified")
}

func TestCancel_NotFoundOrWrongCaller_ReturnsForbidden(t *testing.T) {
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerErr: apperr.ErrNotFound}
	uc := commands.NewCancel(repo, &cancelPaymentFake{})

	_, err := uc.Execute(context.Background(), testutil.MustNewID(), testutil.MustNewID(), "customer")

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrForbidden)
}

func TestCancel_InvalidTransition_ReturnsConflict(t *testing.T) {
	ord := &order.Order{
		ID: testutil.MustNewID(), Status: order.StatusConfirmed,
		Subtotal: decimal.NewFromInt(10), Total: decimal.NewFromInt(10), Currency: "EUR",
	}
	repo := &testutil.MockOrderRepository{FindByIDAndCustomerResult: ord}
	uc := commands.NewCancel(repo, &cancelPaymentFake{})

	_, err := uc.Execute(context.Background(), ord.ID, testutil.MustNewID(), "customer")

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrConflict)
	assert.Empty(t, repo.Updated)
}
