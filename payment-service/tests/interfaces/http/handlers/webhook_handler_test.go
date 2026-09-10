package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-service/internal/application/payment/commands"
	"payment-service/internal/domain/payment"
	"payment-service/internal/infrastructure/persistence"
	"payment-service/internal/interfaces/http/handlers"
	"payment-service/internal/interfaces/http/routes"
	"payment-service/tests/testutil"
)

type fakeWebhookGateway struct {
	statusResult payment.PaymentStatusResult
}

func (f *fakeWebhookGateway) CreatePayment(
	_ context.Context,
	_ payment.CreatePaymentRequest,
) (payment.CreatePaymentResult, error) {
	return payment.CreatePaymentResult{}, nil
}

func (f *fakeWebhookGateway) CancelPayment(_ context.Context, _ string) error {
	return nil
}

func (f *fakeWebhookGateway) GetStatus(
	_ context.Context,
	_ string,
) (payment.PaymentStatusResult, error) {
	return f.statusResult, nil
}

func setupRouter(t *testing.T, gw *fakeWebhookGateway) (*gin.Engine, payment.PaymentRepository) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TablePayment, testutil.TableOutboxEvent)

	repo := persistence.NewPaymentRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)

	handleMollieWebhook := commands.NewHandleMollieWebhook(db.DB, repo, gw, outboxRepo)
	webhookHandler := handlers.NewWebhookHandler(handleMollieWebhook)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	routes.SetupRoutes(router, &routes.Handlers{WebhookHandler: webhookHandler})

	return router, repo
}

func postWebhook(router *gin.Engine, id string) *httptest.ResponseRecorder {
	form := url.Values{}
	if id != "" {
		form.Set("id", id)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/mollie", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestWebhookHandler_MissingID_ReturnsBadRequest(t *testing.T) {
	router, _ := setupRouter(t, &fakeWebhookGateway{})

	w := postWebhook(router, "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookHandler_UnknownPayment_ReturnsOK(t *testing.T) {
	router, _ := setupRouter(t, &fakeWebhookGateway{})

	w := postWebhook(router, "tr_unknown")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookHandler_KnownPayment_UpdatesAndReturnsOK(t *testing.T) {
	gw := &fakeWebhookGateway{statusResult: payment.PaymentStatusResult{Status: payment.StatusSucceeded}}
	router, repo := setupRouter(t, gw)

	p := payment.NewPayment(
		testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(), testutil.MustNewID(),
		"order", decimal.NewFromFloat(24.50), "EUR", decimal.Zero, "mollie",
	)
	require.NoError(t, repo.Create(context.Background(), p))
	p.AttachGatewayReference("tr_known")
	require.NoError(t, repo.Update(context.Background(), p))

	w := postWebhook(router, "tr_known")

	assert.Equal(t, http.StatusOK, w.Code)

	found, err := repo.FindByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, payment.StatusSucceeded, found.Status)
}
