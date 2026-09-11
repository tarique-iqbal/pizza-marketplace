package payment

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order-service/internal/domain/order"
	"order-service/internal/infrastructure/payment/pb"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.PaymentServiceClient
}

func Dial(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial payment-service: %w", err)
	}

	return conn, nil
}

func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{conn: conn, client: pb.NewPaymentServiceClient(conn)}
}

func (c *Client) CreatePayment(
	ctx context.Context,
	req order.CreatePaymentRequest,
) (order.CreatePaymentResult, error) {
	resp, err := c.client.CreatePayment(ctx, &pb.CreatePaymentRequest{
		SubjectType:  "order",
		SubjectId:    req.OrderID.String(),
		RestaurantId: req.RestaurantID.String(),
		CustomerId:   req.CustomerID.String(),
		Amount:       req.Amount.StringFixed(2),
		Currency:     req.Currency,
		RedirectUrl:  req.RedirectURL,
	})
	if err != nil {
		return order.CreatePaymentResult{}, fmt.Errorf("%w: %s", order.ErrPaymentServiceUnavailable, err)
	}

	return order.CreatePaymentResult{
		PaymentID:   resp.GetPaymentId(),
		CheckoutURL: resp.GetCheckoutUrl(),
	}, nil
}

func (c *Client) CancelPayment(ctx context.Context, paymentID string) error {
	if _, err := c.client.CancelPayment(ctx, &pb.CancelPaymentRequest{PaymentId: paymentID}); err != nil {
		return fmt.Errorf("%w: %s", order.ErrPaymentServiceUnavailable, err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
