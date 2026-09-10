package commands_test

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentapp "payment-service/internal/application/payment"
	"payment-service/internal/application/payment/commands"
	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/persistence"
	"payment-service/tests/testutil"
)

type fakeGateway struct {
	createResult payment.CreatePaymentResult
	createErr    error
	statusResult payment.PaymentStatusResult
	statusErr    error
	cancelErr    error

	createCalls []payment.CreatePaymentRequest
	cancelCalls []string
}

func (f *fakeGateway) CreatePayment(
	_ context.Context,
	req payment.CreatePaymentRequest,
) (payment.CreatePaymentResult, error) {
	f.createCalls = append(f.createCalls, req)
	return f.createResult, f.createErr
}

func (f *fakeGateway) CancelPayment(_ context.Context, gatewayPaymentID string) error {
	f.cancelCalls = append(f.cancelCalls, gatewayPaymentID)
	return f.cancelErr
}

func (f *fakeGateway) GetStatus(
	_ context.Context,
	_ string,
) (payment.PaymentStatusResult, error) {
	return f.statusResult, f.statusErr
}

func newCreatePaymentRequest() paymentapp.CreatePaymentRequest {
	return paymentapp.CreatePaymentRequest{
		SubjectType:  "order",
		SubjectID:    testutil.MustNewID(),
		RestaurantID: testutil.MustNewID(),
		CustomerID:   testutil.MustNewID(),
		Amount:       decimal.NewFromFloat(24.50),
		Currency:     "EUR",
		RedirectURL:  "https://frontend.example/orders/abc",
	}
}

func TestCreatePayment_FreshRequest_CreatesAndAttachesGatewayReference(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)
	gw := &fakeGateway{
		createResult: payment.CreatePaymentResult{
			GatewayPaymentID: "tr_abc123",
			CheckoutURL:      "https://mollie.test/checkout/tr_abc123",
		},
	}
	uc := commands.NewCreatePayment(repo, gw, "https://payment-service.internal")

	req := newCreatePaymentRequest()

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.PaymentID)
	assert.Equal(t, "https://mollie.test/checkout/tr_abc123", resp.CheckoutURL)
	assert.Equal(t, string(payment.StatusPending), resp.Status)

	require.Len(t, gw.createCalls, 1)
	assert.Equal(t, "https://payment-service.internal/webhooks/mollie", gw.createCalls[0].WebhookURL)

	found, err := repo.FindByID(context.Background(), resp.PaymentID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.GatewayPaymentID)
	assert.Equal(t, "tr_abc123", *found.GatewayPaymentID)
}

func TestCreatePayment_SameSubjectTwice_IsIdempotent(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)
	gw := &fakeGateway{
		createResult: payment.CreatePaymentResult{
			GatewayPaymentID: "tr_abc123",
			CheckoutURL:      "https://mollie.test/checkout/tr_abc123",
		},
		statusResult: payment.PaymentStatusResult{
			Status:      payment.StatusPending,
			CheckoutURL: "https://mollie.test/checkout/tr_abc123",
		},
	}
	uc := commands.NewCreatePayment(repo, gw, "https://payment-service.internal")

	req := newCreatePaymentRequest()

	first, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)

	second, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, first.PaymentID, second.PaymentID)
	assert.Equal(t, "https://mollie.test/checkout/tr_abc123", second.CheckoutURL)

	// only the first call reached the gateway's CreatePayment
	assert.Len(t, gw.createCalls, 1)

	var count int64
	require.NoError(t, db.DB.Model(&payment.Payment{}).
		Where("subject_type = ? AND subject_id = ?", req.SubjectType, req.SubjectID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestCreatePayment_ExistingRowWithoutGatewayReference_SelfHeals(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)
	req := newCreatePaymentRequest()

	// simulate a prior attempt that persisted the row but crashed before reaching the gateway
	existing := payment.NewPayment(
		testutil.MustNewID(), req.SubjectID, req.RestaurantID, req.CustomerID,
		req.SubjectType, req.Amount, req.Currency, decimal.Zero, "mollie",
	)
	require.NoError(t, repo.Create(context.Background(), existing))

	gw := &fakeGateway{
		createResult: payment.CreatePaymentResult{
			GatewayPaymentID: "tr_healed",
			CheckoutURL:      "https://mollie.test/checkout/tr_healed",
		},
	}
	uc := commands.NewCreatePayment(repo, gw, "https://payment-service.internal")

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, existing.ID, resp.PaymentID)
	assert.Equal(t, "https://mollie.test/checkout/tr_healed", resp.CheckoutURL)
	require.Len(t, gw.createCalls, 1)

	var count int64
	require.NoError(t, db.DB.Model(&payment.Payment{}).
		Where("subject_type = ? AND subject_id = ?", req.SubjectType, req.SubjectID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
