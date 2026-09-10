package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"payment-service/internal/application/payment/commands"
	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/persistence"
	grpcserver "payment-service/internal/interfaces/grpc"
	"payment-service/internal/interfaces/grpc/pb"
	"payment-service/tests/testutil"
)

type fakeGateway struct {
	createResult payment.CreatePaymentResult
	createErr    error
}

func (f *fakeGateway) CreatePayment(
	_ context.Context,
	_ payment.CreatePaymentRequest,
) (payment.CreatePaymentResult, error) {
	return f.createResult, f.createErr
}

func (f *fakeGateway) CancelPayment(_ context.Context, _ string) error {
	return nil
}

func (f *fakeGateway) GetStatus(
	_ context.Context,
	_ string,
) (payment.PaymentStatusResult, error) {
	return payment.PaymentStatusResult{}, nil
}

func newTestServer(t *testing.T, gw *fakeGateway) *grpcserver.Server {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment)

	repo := persistence.NewPaymentRepository(db.DB)
	createPayment := commands.NewCreatePayment(repo, gw, "https://payment-service.internal")
	cancelPayment := commands.NewCancelPayment(repo, gw)

	return grpcserver.NewServer(createPayment, cancelPayment)
}

func TestServer_CreatePayment_Success(t *testing.T) {
	gw := &fakeGateway{
		createResult: payment.CreatePaymentResult{
			GatewayPaymentID: "tr_abc123",
			CheckoutURL:      "https://mollie.test/checkout/tr_abc123",
		},
	}
	srv := newTestServer(t, gw)

	resp, err := srv.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		SubjectType:  "order",
		SubjectId:    uuid.New().String(),
		RestaurantId: uuid.New().String(),
		CustomerId:   uuid.New().String(),
		Amount:       "24.50",
		Currency:     "EUR",
		RedirectUrl:  "https://frontend.example/orders/abc",
	})
	require.NoError(t, err)

	assert.NotEmpty(t, resp.GetPaymentId())
	assert.Equal(t, "https://mollie.test/checkout/tr_abc123", resp.GetCheckoutUrl())
	assert.Equal(t, string(payment.StatusPending), resp.GetStatus())
}

func TestServer_CreatePayment_InvalidSubjectID_ReturnsInvalidArgument(t *testing.T) {
	srv := newTestServer(t, &fakeGateway{})

	_, err := srv.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		SubjectType:  "order",
		SubjectId:    "not-a-uuid",
		RestaurantId: uuid.New().String(),
		CustomerId:   uuid.New().String(),
		Amount:       "24.50",
		Currency:     "EUR",
		RedirectUrl:  "https://frontend.example/orders/abc",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_CreatePayment_InvalidAmount_ReturnsInvalidArgument(t *testing.T) {
	srv := newTestServer(t, &fakeGateway{})

	_, err := srv.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		SubjectType:  "order",
		SubjectId:    uuid.New().String(),
		RestaurantId: uuid.New().String(),
		CustomerId:   uuid.New().String(),
		Amount:       "not-a-number",
		Currency:     "EUR",
		RedirectUrl:  "https://frontend.example/orders/abc",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_CreatePayment_GatewayError_ReturnsInternal(t *testing.T) {
	gw := &fakeGateway{createErr: errors.New("mollie unreachable")}
	srv := newTestServer(t, gw)

	_, err := srv.CreatePayment(context.Background(), &pb.CreatePaymentRequest{
		SubjectType:  "order",
		SubjectId:    uuid.New().String(),
		RestaurantId: uuid.New().String(),
		CustomerId:   uuid.New().String(),
		Amount:       "24.50",
		Currency:     "EUR",
		RedirectUrl:  "https://frontend.example/orders/abc",
	})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestServer_CancelPayment_InvalidPaymentID_ReturnsInvalidArgument(t *testing.T) {
	srv := newTestServer(t, &fakeGateway{})

	_, err := srv.CancelPayment(context.Background(), &pb.CancelPaymentRequest{
		PaymentId: "not-a-uuid",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServer_CancelPayment_UnknownPayment_Succeeds(t *testing.T) {
	srv := newTestServer(t, &fakeGateway{})

	resp, err := srv.CancelPayment(context.Background(), &pb.CancelPaymentRequest{
		PaymentId: uuid.New().String(),
	})
	require.NoError(t, err)
	assert.Equal(t, "canceled", resp.GetStatus())
}
