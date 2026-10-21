package container

import (
	"os"

	"payment-service/internal/application/payment/commands"
	"payment-service/internal/application/payment/queries"
	"payment-service/internal/infrastructure/db"
	"payment-service/internal/infrastructure/gateway"
	"payment-service/internal/infrastructure/persistence"
	grpcserver "payment-service/internal/interfaces/grpc"
	"payment-service/internal/interfaces/http/handlers"
	"payment-service/internal/interfaces/http/routes"

	"pizza-marketplace/shared/health"
)

type APIContainer struct {
	*Shared
	GRPCServer *grpcserver.Server
	Handlers   *routes.Handlers
}

func NewAPIContainer() (*APIContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	paymentRepo := persistence.NewPaymentRepository(base.DB)
	mollieGateway := gateway.NewMollieGateway(os.Getenv("MOLLIE_API_KEY"), gateway.MollieBaseURL)
	publicBaseURL := os.Getenv("PUBLIC_BASE_URL")

	createPayment := commands.NewCreatePayment(paymentRepo, mollieGateway, publicBaseURL)
	cancelPayment := commands.NewCancelPayment(paymentRepo, mollieGateway)
	handleMollieWebhook := commands.NewHandleMollieWebhook(base.DB, paymentRepo, mollieGateway, base.OutboxRepo)
	getPaymentStatus := queries.NewGetPaymentStatus(paymentRepo)

	grpcServer := grpcserver.NewServer(createPayment, cancelPayment, getPaymentStatus)
	webhookHandler := handlers.NewWebhookHandler(handleMollieWebhook)

	// health
	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: &db.Postgres{DB: base.DB}},
	)
	healthHandler := handlers.NewHealthHandler(readiness)

	return &APIContainer{
		Shared:     base,
		GRPCServer: grpcServer,
		Handlers:   &routes.Handlers{WebhookHandler: webhookHandler, HealthHandler: healthHandler},
	}, nil
}

func (c *APIContainer) Close() {
	if c.DB != nil {
		db, err := c.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}
}
