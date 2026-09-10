package container

import (
	"os"

	"payment-service/internal/application/payment/commands"
	"payment-service/internal/infrastructure/gateway"
	"payment-service/internal/infrastructure/persistence"
	grpcserver "payment-service/internal/interfaces/grpc"
)

type APIContainer struct {
	*Shared
	GRPCServer *grpcserver.Server
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

	grpcServer := grpcserver.NewServer(createPayment, cancelPayment)

	return &APIContainer{
		Shared:     base,
		GRPCServer: grpcServer,
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
