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

func (uc *CreatePayment) Execute(
	ctx context.Context,
	req paymentapp.CreatePaymentRequest,
) (paymentapp.CreatePaymentResponse, error) {
	existing, err := uc.repo.FindBySubject(ctx, req.SubjectType, req.SubjectID)
	if err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to look up existing payment: %w", err)
	}

	if existing != nil {
		return uc.handleExisting(ctx, existing, req)
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

	if err := uc.repo.Create(ctx, p); err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to create payment: %w", err)
	}

	return uc.callGatewayAndAttach(ctx, p, req)
}

func (uc *CreatePayment) handleExisting(
	ctx context.Context,
	existing *payment.Payment,
	req paymentapp.CreatePaymentRequest,
) (paymentapp.CreatePaymentResponse, error) {
	if existing.GatewayPaymentID == nil {
		return uc.callGatewayAndAttach(ctx, existing, req)
	}

	result, err := uc.gateway.GetStatus(ctx, *existing.GatewayPaymentID)
	if err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to check payment status: %w", err)
	}

	return paymentapp.CreatePaymentResponse{
		PaymentID:   existing.ID,
		CheckoutURL: result.CheckoutURL,
		Status:      string(existing.Status),
	}, nil
}

func (uc *CreatePayment) callGatewayAndAttach(
	ctx context.Context,
	p *payment.Payment,
	req paymentapp.CreatePaymentRequest,
) (paymentapp.CreatePaymentResponse, error) {
	result, err := uc.gateway.CreatePayment(ctx, payment.CreatePaymentRequest{
		RestaurantID: req.RestaurantID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		RedirectURL:  req.RedirectURL,
		WebhookURL:   uc.publicBaseURL + "/webhooks/mollie",
	})
	if err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to create gateway payment: %w", err)
	}

	p.AttachGatewayReference(result.GatewayPaymentID)

	if err := uc.repo.Update(ctx, p); err != nil {
		return paymentapp.CreatePaymentResponse{}, fmt.Errorf("failed to record gateway reference: %w", err)
	}

	return paymentapp.CreatePaymentResponse{
		PaymentID:   p.ID,
		CheckoutURL: result.CheckoutURL,
		Status:      string(p.Status),
	}, nil
}
