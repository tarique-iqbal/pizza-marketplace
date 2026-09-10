package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"payment-service/internal/application/payment/commands"
	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/persistence"
	"payment-service/tests/testutil"
)

func TestCancelPayment_SwallowsGatewayError(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)

	p := payment.NewPayment(
		testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(),
		"order", decimal.NewFromFloat(24.50), "EUR", decimal.Zero, "mollie",
	)
	p.AttachGatewayReference("tr_already_paid")
	require.NoError(t, repo.Create(context.Background(), p))

	gw := &fakeGateway{cancelErr: errors.New("mollie: payment can no longer be canceled")}
	uc := commands.NewCancelPayment(repo, gw)

	err := uc.Execute(context.Background(), p.ID)
	require.NoError(t, err)
	require.Len(t, gw.cancelCalls, 1)
}

func TestCancelPayment_NoGatewayReference_NoOp(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)

	p := payment.NewPayment(
		testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(),
		"order", decimal.NewFromFloat(24.50), "EUR", decimal.Zero, "mollie",
	)
	require.NoError(t, repo.Create(context.Background(), p))

	gw := &fakeGateway{}
	uc := commands.NewCancelPayment(repo, gw)

	err := uc.Execute(context.Background(), p.ID)
	require.NoError(t, err)
	require.Empty(t, gw.cancelCalls)
}

func TestCancelPayment_PaymentNotFound_NoOp(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)
	gw := &fakeGateway{}
	uc := commands.NewCancelPayment(repo, gw)

	err := uc.Execute(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Empty(t, gw.cancelCalls)
}
