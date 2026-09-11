package payment_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"order-service/internal/domain/order"
	"order-service/internal/infrastructure/payment"
	"order-service/internal/infrastructure/payment/pb"
)

type fakeServer struct {
	pb.UnimplementedPaymentServiceServer
	createReq  *pb.CreatePaymentRequest
	createResp *pb.CreatePaymentResponse
	createErr  error
	cancelReq  *pb.CancelPaymentRequest
	cancelErr  error
}

func (s *fakeServer) CreatePayment(
	_ context.Context,
	req *pb.CreatePaymentRequest,
) (*pb.CreatePaymentResponse, error) {
	s.createReq = req
	if s.createErr != nil {
		return nil, s.createErr
	}

	return s.createResp, nil
}

func (s *fakeServer) CancelPayment(
	_ context.Context,
	req *pb.CancelPaymentRequest,
) (*pb.CancelPaymentResponse, error) {
	s.cancelReq = req
	if s.cancelErr != nil {
		return nil, s.cancelErr
	}

	return &pb.CancelPaymentResponse{Status: "canceled"}, nil
}

func newTestClient(t *testing.T, srv *fakeServer) *payment.Client {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterPaymentServiceServer(grpcServer, srv)

	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return payment.NewClient(conn)
}

func TestClient_CreatePayment_MapsFieldsBothWays(t *testing.T) {
	srv := &fakeServer{
		createResp: &pb.CreatePaymentResponse{
			PaymentId:   "11111111-1111-1111-1111-111111111111",
			CheckoutUrl: "https://mollie.test/checkout/tr_abc123",
			Status:      "pending",
		},
	}
	client := newTestClient(t, srv)

	orderID := uuid.New()
	restaurantID := uuid.New()
	customerID := uuid.New()

	result, err := client.CreatePayment(context.Background(), order.CreatePaymentRequest{
		OrderID:      orderID,
		RestaurantID: restaurantID,
		CustomerID:   customerID,
		Amount:       decimal.NewFromFloat(24.5),
		Currency:     "EUR",
		RedirectURL:  "https://frontend.test/orders/" + orderID.String(),
	})
	require.NoError(t, err)

	require.NotNil(t, srv.createReq)
	assert.Equal(t, "order", srv.createReq.GetSubjectType())
	assert.Equal(t, orderID.String(), srv.createReq.GetSubjectId())
	assert.Equal(t, restaurantID.String(), srv.createReq.GetRestaurantId())
	assert.Equal(t, customerID.String(), srv.createReq.GetCustomerId())
	assert.Equal(t, "24.50", srv.createReq.GetAmount())
	assert.Equal(t, "EUR", srv.createReq.GetCurrency())

	assert.Equal(t, "11111111-1111-1111-1111-111111111111", result.PaymentID)
	assert.Equal(t, "https://mollie.test/checkout/tr_abc123", result.CheckoutURL)
}

func TestClient_CreatePayment_PropagatesGRPCError(t *testing.T) {
	srv := &fakeServer{createErr: status.Error(codes.Internal, "mollie unavailable")}
	client := newTestClient(t, srv)

	_, err := client.CreatePayment(context.Background(), order.CreatePaymentRequest{
		OrderID:      uuid.New(),
		RestaurantID: uuid.New(),
		CustomerID:   uuid.New(),
		Amount:       decimal.NewFromInt(10),
		Currency:     "EUR",
		RedirectURL:  "https://frontend.test",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, order.ErrPaymentServiceUnavailable))
}

func TestClient_CancelPayment_MapsFieldsBothWays(t *testing.T) {
	srv := &fakeServer{}
	client := newTestClient(t, srv)

	err := client.CancelPayment(context.Background(), "pay_123")
	require.NoError(t, err)

	require.NotNil(t, srv.cancelReq)
	assert.Equal(t, "pay_123", srv.cancelReq.GetPaymentId())
}

func TestClient_CancelPayment_PropagatesGRPCError(t *testing.T) {
	srv := &fakeServer{cancelErr: status.Error(codes.Internal, "mollie unavailable")}
	client := newTestClient(t, srv)

	err := client.CancelPayment(context.Background(), "pay_123")

	require.Error(t, err)
	assert.True(t, errors.Is(err, order.ErrPaymentServiceUnavailable))
}
