package queries_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-service/internal/application/payment/queries"
	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/persistence"
	apperr "payment-service/internal/shared/errors"
	"payment-service/tests/testutil"
)

func TestGetPaymentStatus_ReturnsCurrentStatus(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)

	p := payment.NewPayment(
		testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(),
		"order", decimal.NewFromFloat(24.50), "EUR", decimal.Zero, "mollie",
	)
	require.NoError(t, repo.Create(context.Background(), p))
	require.NoError(t, p.MarkSucceeded())
	require.NoError(t, repo.Update(context.Background(), p))

	qry := queries.NewGetPaymentStatus(repo)

	status, err := qry.Execute(context.Background(), p.ID)

	require.NoError(t, err)
	assert.Equal(t, payment.StatusSucceeded, status)
}

func TestGetPaymentStatus_NotFound(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)
	qry := queries.NewGetPaymentStatus(repo)

	_, err := qry.Execute(context.Background(), testutil.MustNewID())

	require.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotFound)
}
