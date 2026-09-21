package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	paymentapp "payment-service/internal/application/payment"
	"payment-service/internal/domain/payment"
)

type CreatePayment struct {
	repo          payment.PaymentRepository
	gateway       payment.PaymentGateway
	publicBaseURL string
}

func NewCreatePayment(
	repo payment.PaymentRepository,
	gateway payment.PaymentGateway,
	publicBaseURL string,
) *CreatePayment {
	return &CreatePayment{
		repo:          repo,
		gateway:       gateway,
		publicBaseURL: publicBaseURL,
	}
}

func (cmd *CreatePayment) Execute(
	ctx context.Context,
	req paymentapp.CreatePaymentRequest,
) (paymentapp.CreatePaymentResponse, error) {
	existing, err := cmd.repo.FindBySubject(ctx, req.SubjectType, req.SubjectID)
	if err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to look up existing payment: %w", err)
	}

	if existing != nil {
		return cmd.handleExisting(ctx, existing, req)
	}

	p := payment.NewPayment(
		uuid.Must(uuid.NewV7()),
		req.SubjectID,
		req.RestaurantID,
		req.CustomerID,
		req.SubjectType,
		req.Amount,
		req.Currency,
		decimal.Zero,
		"mollie",
	)

	if err := cmd.repo.Create(ctx, p); err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to create payment: %w", err)
	}

	return cmd.callGatewayAndAttach(ctx, p, req)
}

func (cmd *CreatePayment) handleExisting(
	ctx context.Context,
	existing *payment.Payment,
	req paymentapp.CreatePaymentRequest,
) (paymentapp.CreatePaymentResponse, error) {
	if existing.GatewayPaymentID == nil {
		return cmd.callGatewayAndAttach(ctx, existing, req)
	}

	result, err := cmd.gateway.GetStatus(ctx, *existing.GatewayPaymentID)
	if err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to check payment status: %w", err)
	}

	return paymentapp.CreatePaymentResponse{
		PaymentID:   existing.ID,
		CheckoutURL: result.CheckoutURL,
		Status:      string(existing.Status),
	}, nil
}

func (cmd *CreatePayment) callGatewayAndAttach(
	ctx context.Context,
	p *payment.Payment,
	req paymentapp.CreatePaymentRequest,
) (paymentapp.CreatePaymentResponse, error) {
	result, err := cmd.gateway.CreatePayment(ctx, payment.CreatePaymentRequest{
		RestaurantID: req.RestaurantID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		RedirectURL:  req.RedirectURL,
		WebhookURL:   cmd.publicBaseURL + "/webhooks/mollie",
	})
	if err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to create gateway payment: %w", err)
	}

	p.AttachGatewayReference(result.GatewayPaymentID)

	if err := cmd.repo.Update(ctx, p); err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to record gateway reference: %w", err)
	}

	return paymentapp.CreatePaymentResponse{
		PaymentID:   p.ID,
		CheckoutURL: result.CheckoutURL,
		Status:      string(p.Status),
	}, nil
}
