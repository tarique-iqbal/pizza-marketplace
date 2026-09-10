package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"payment-service/internal/domain/payment"
)

const MollieBaseURL = "https://api.mollie.com/v2"

type MollieGateway struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewMollieGateway(apiKey, baseURL string) *MollieGateway {
	return &MollieGateway{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type mollieAmount struct {
	Currency string `json:"currency"`
	Value    string `json:"value"`
}

type molliePaymentRequest struct {
	Amount      mollieAmount `json:"amount"`
	Description string       `json:"description"`
	RedirectURL string       `json:"redirectUrl"`
	WebhookURL  string       `json:"webhookUrl,omitempty"`
}

type molliePaymentResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Links  struct {
		Checkout struct {
			Href string `json:"href"`
		} `json:"checkout"`
	} `json:"_links"`
}

func (g *MollieGateway) CreatePayment(
	ctx context.Context,
	req payment.CreatePaymentRequest,
) (payment.CreatePaymentResult, error) {
	body := molliePaymentRequest{
		Amount: mollieAmount{
			Currency: req.Currency,
			Value:    req.Amount.StringFixed(2),
		},
		Description: fmt.Sprintf("Payment for restaurant %s", req.RestaurantID),
		RedirectURL: req.RedirectURL,
		WebhookURL:  req.WebhookURL,
	}

	var resp molliePaymentResponse
	if err := g.doJSON(ctx, http.MethodPost, "/payments", body, &resp); err != nil {
		return payment.CreatePaymentResult{}, err
	}

	return payment.CreatePaymentResult{
		GatewayPaymentID: resp.ID,
		CheckoutURL:      resp.Links.Checkout.Href,
	}, nil
}

func (g *MollieGateway) CancelPayment(ctx context.Context, gatewayPaymentID string) error {
	return g.doJSON(ctx, http.MethodDelete, "/payments/"+gatewayPaymentID, nil, nil)
}

func (g *MollieGateway) GetStatus(
	ctx context.Context,
	gatewayPaymentID string,
) (payment.PaymentStatusResult, error) {
	var resp molliePaymentResponse
	if err := g.doJSON(ctx, http.MethodGet, "/payments/"+gatewayPaymentID, nil, &resp); err != nil {
		return payment.PaymentStatusResult{}, err
	}

	status, reason := mapMollieStatus(resp.Status)

	return payment.PaymentStatusResult{Status: status, Reason: reason}, nil
}

func mapMollieStatus(mollieStatus string) (payment.PaymentStatus, string) {
	switch mollieStatus {
	case "paid":
		return payment.StatusSucceeded, ""
	case "canceled", "expired", "failed":
		return payment.StatusFailed, "payment " + mollieStatus
	default:
		return payment.StatusPending, ""
	}
}

func (g *MollieGateway) doJSON(ctx context.Context, method, path string, body, out any) error {
	var reqBody bytes.Reader

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reqBody = *bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, &reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mollie request failed: status=%s", resp.Status)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
