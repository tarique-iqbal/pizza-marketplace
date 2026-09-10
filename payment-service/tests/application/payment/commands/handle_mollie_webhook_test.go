package commands_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentapp "payment-service/internal/application/payment"
	"payment-service/internal/application/payment/commands"
	"payment-service/internal/domain/outbox"
	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/persistence"
	"payment-service/tests/testutil"
)

func newPendingPayment(t *testing.T, repo payment.PaymentRepository, gatewayPaymentID string) *payment.Payment {
	t.Helper()

	p := payment.NewPayment(
		testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(),
		"order", decimal.NewFromFloat(24.50), "EUR", decimal.Zero, "mollie",
	)
	require.NoError(t, repo.Create(context.Background(), p))

	p.AttachGatewayReference(gatewayPaymentID)
	require.NoError(t, repo.Update(context.Background(), p))

	return p
}

func countOutboxEvents(t *testing.T, db *testutil.TestDB) int64 {
	t.Helper()

	var count int64
	require.NoError(t, db.DB.Model(&outbox.OutboxEvent{}).Count(&count).Error)
	return count
}

func TestHandleMollieWebhook_UnknownPayment_NoOp(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment, testutil.TableOutboxEvent)

	repo := persistence.NewPaymentRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	gw := &fakeGateway{}
	uc := commands.NewHandleMollieWebhook(db.DB, repo, gw, outboxRepo)

	err := uc.Execute(context.Background(), "tr_unknown")
	require.NoError(t, err)

	assert.Equal(t, int64(0), countOutboxEvents(t, db))
}

func TestHandleMollieWebhook_AlreadyTerminalPayment_NoOp(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment, testutil.TableOutboxEvent)

	repo := persistence.NewPaymentRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	p := newPendingPayment(t, repo, "tr_terminal")
	require.NoError(t, p.MarkSucceeded())
	require.NoError(t, repo.Update(context.Background(), p))

	gw := &fakeGateway{statusResult: payment.PaymentStatusResult{Status: payment.StatusSucceeded}}
	uc := commands.NewHandleMollieWebhook(db.DB, repo, gw, outboxRepo)

	err := uc.Execute(context.Background(), "tr_terminal")
	require.NoError(t, err)

	assert.Equal(t, int64(0), countOutboxEvents(t, db))
}

func TestHandleMollieWebhook_NonTerminalStatus_NoOp(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment, testutil.TableOutboxEvent)

	repo := persistence.NewPaymentRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	p := newPendingPayment(t, repo, "tr_open")

	gw := &fakeGateway{statusResult: payment.PaymentStatusResult{Status: payment.StatusPending}}
	uc := commands.NewHandleMollieWebhook(db.DB, repo, gw, outboxRepo)

	err := uc.Execute(context.Background(), "tr_open")
	require.NoError(t, err)

	assert.Equal(t, int64(0), countOutboxEvents(t, db))

	found, err := repo.FindByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, payment.StatusPending, found.Status)
}

func TestHandleMollieWebhook_Paid_MarksSucceededAndPublishes(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment, testutil.TableOutboxEvent)

	repo := persistence.NewPaymentRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	p := newPendingPayment(t, repo, "tr_paid")

	gw := &fakeGateway{statusResult: payment.PaymentStatusResult{Status: payment.StatusSucceeded}}
	uc := commands.NewHandleMollieWebhook(db.DB, repo, gw, outboxRepo)

	err := uc.Execute(context.Background(), "tr_paid")
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, payment.StatusSucceeded, found.Status)

	assert.Equal(t, int64(1), countOutboxEvents(t, db))

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)
	assert.Equal(t, "payment.succeeded", event.EventName)

	var decoded paymentapp.PaymentSucceededPayload
	require.NoError(t, json.Unmarshal(event.Payload, &decoded))
	assert.Equal(t, p.ID, decoded.PaymentID)
	assert.Equal(t, "24.50", decoded.Amount)
}

func TestHandleMollieWebhook_Failed_MarksFailedAndPublishes(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment, testutil.TableOutboxEvent)

	repo := persistence.NewPaymentRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	p := newPendingPayment(t, repo, "tr_expired")

	gw := &fakeGateway{
		statusResult: payment.PaymentStatusResult{Status: payment.StatusFailed, Reason: "payment expired"},
	}
	uc := commands.NewHandleMollieWebhook(db.DB, repo, gw, outboxRepo)

	err := uc.Execute(context.Background(), "tr_expired")
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, payment.StatusFailed, found.Status)
	require.NotNil(t, found.FailureReason)
	assert.Equal(t, "payment expired", *found.FailureReason)

	assert.Equal(t, int64(1), countOutboxEvents(t, db))

	var event outbox.OutboxEvent
	require.NoError(t, db.DB.First(&event).Error)
	assert.Equal(t, "payment.failed", event.EventName)

	var decoded paymentapp.PaymentFailedPayload
	require.NoError(t, json.Unmarshal(event.Payload, &decoded))
	assert.Equal(t, "payment expired", decoded.Reason)
}
