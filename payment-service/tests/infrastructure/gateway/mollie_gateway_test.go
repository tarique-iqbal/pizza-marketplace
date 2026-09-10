package gateway_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/gateway"
)

func TestMollieGateway_CreatePayment_RequestShape(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotContentType string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "tr_test123",
			"status": "open",
			"_links": map[string]any{
				"checkout": map[string]any{"href": "https://mollie.test/checkout/tr_test123"},
			},
		})
	}))
	defer server.Close()

	gw := gateway.NewMollieGateway("test_apikey", server.URL)

	result, err := gw.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		RestaurantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Amount:       decimal.NewFromFloat(24.5),
		Currency:     "EUR",
		RedirectURL:  "https://frontend.example/orders/abc",
		WebhookURL:   "https://payment-service.internal/webhooks/mollie",
	})
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/payments", gotPath)
	assert.Equal(t, "Bearer test_apikey", gotAuth)
	assert.Equal(t, "application/json", gotContentType)

	amount, _ := gotBody["amount"].(map[string]any)
	assert.Equal(t, "EUR", amount["currency"])
	assert.Equal(t, "24.50", amount["value"])
	assert.Equal(t, "https://frontend.example/orders/abc", gotBody["redirectUrl"])
	assert.Equal(t, "https://payment-service.internal/webhooks/mollie", gotBody["webhookUrl"])

	assert.Equal(t, "tr_test123", result.GatewayPaymentID)
	assert.Equal(t, "https://mollie.test/checkout/tr_test123", result.CheckoutURL)
}

func TestMollieGateway_GetStatus_MapsEveryMollieStatus(t *testing.T) {
	cases := []struct {
		mollieStatus   string
		expectedStatus payment.PaymentStatus
		expectReason   bool
	}{
		{"paid", payment.StatusSucceeded, false},
		{"canceled", payment.StatusFailed, true},
		{"expired", payment.StatusFailed, true},
		{"failed", payment.StatusFailed, true},
		{"open", payment.StatusPending, false},
		{"pending", payment.StatusPending, false},
		{"authorized", payment.StatusPending, false},
	}

	for _, tc := range cases {
		t.Run(tc.mollieStatus, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/payments/tr_test123", r.URL.Path)

				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":     "tr_test123",
					"status": tc.mollieStatus,
				})
			}))
			defer server.Close()

			gw := gateway.NewMollieGateway("test_apikey", server.URL)

			result, err := gw.GetStatus(context.Background(), "tr_test123")
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, result.Status)
			if tc.expectReason {
				assert.NotEmpty(t, result.Reason)
			} else {
				assert.Empty(t, result.Reason)
			}
		})
	}
}

func TestMollieGateway_CancelPayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/payments/tr_test123", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "tr_test123", "status": "canceled"})
	}))
	defer server.Close()

	gw := gateway.NewMollieGateway("test_apikey", server.URL)

	err := gw.CancelPayment(context.Background(), "tr_test123")
	require.NoError(t, err)
}

func TestMollieGateway_NonSuccessResponse_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	gw := gateway.NewMollieGateway("test_apikey", server.URL)

	err := gw.CancelPayment(context.Background(), "tr_already_paid")
	require.Error(t, err)
}
