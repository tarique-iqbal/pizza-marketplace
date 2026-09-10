package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentapp "payment-service/internal/application/payment"
	"payment-service/internal/application/payment/commands"
	"payment-service/internal/interfaces/grpc/pb"
)

type Server struct {
	pb.UnimplementedPaymentServiceServer
	createPayment *commands.CreatePayment
	cancelPayment *commands.CancelPayment
}

func NewServer(createPayment *commands.CreatePayment, cancelPayment *commands.CancelPayment) *Server {
	return &Server{
		createPayment: createPayment,
		cancelPayment: cancelPayment,
	}
}

func (s *Server) CreatePayment(
	ctx context.Context,
	req *pb.CreatePaymentRequest,
) (*pb.CreatePaymentResponse, error) {
	subjectID, err := uuid.Parse(req.GetSubjectId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid subject_id: %v", err)
	}

	restaurantID, err := uuid.Parse(req.GetRestaurantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid restaurant_id: %v", err)
	}

	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid customer_id: %v", err)
	}

	amount, err := decimal.NewFromString(req.GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid amount: %v", err)
	}

	resp, err := s.createPayment.Execute(ctx, paymentapp.CreatePaymentRequest{
		SubjectType:  req.GetSubjectType(),
		SubjectID:    subjectID,
		RestaurantID: restaurantID,
		CustomerID:   customerID,
		Amount:       amount,
		Currency:     req.GetCurrency(),
		RedirectURL:  req.GetRedirectUrl(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create payment: %v", err)
	}

	return &pb.CreatePaymentResponse{
		PaymentId:   resp.PaymentID.String(),
		CheckoutUrl: resp.CheckoutURL,
		Status:      resp.Status,
	}, nil
}

func (s *Server) CancelPayment(
	ctx context.Context,
	req *pb.CancelPaymentRequest,
) (*pb.CancelPaymentResponse, error) {
	paymentID, err := uuid.Parse(req.GetPaymentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment_id: %v", err)
	}

	if err := s.cancelPayment.Execute(ctx, paymentID); err != nil {
		return nil, status.Errorf(codes.Internal, "cancel payment: %v", err)
	}

	return &pb.CancelPaymentResponse{Status: "canceled"}, nil
}
