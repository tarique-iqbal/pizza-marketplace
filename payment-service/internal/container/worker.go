package container

import (
	"log/slog"

	outboxapp "payment-service/internal/application/outbox"
	"payment-service/internal/infrastructure/db"
	"payment-service/internal/infrastructure/messaging"
	"payment-service/internal/interfaces/http/handlers"

	"pizza-marketplace/shared/health"
)

type WorkerContainer struct {
	*Shared
	Publisher     *messaging.RabbitMQPublisher
	OutboxWorker  *outboxapp.Worker
	HealthHandler *handlers.HealthHandler
}

func NewWorkerContainer(logger *slog.Logger) (*WorkerContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	publisher, err := messaging.NewRabbitMQPublisher(base.AMQPURL)
	if err != nil {
		return nil, err
	}

	relayer := outboxapp.NewRelay(publisher)
	outboxWorker := outboxapp.NewWorker(base.OutboxRepo, relayer, outboxapp.DefaultConfig(), logger)

	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: &db.Postgres{DB: base.DB}},
		health.Check{Name: "rabbitmq", Pinger: publisher},
	)
	healthHandler := handlers.NewHealthHandler(readiness)

	return &WorkerContainer{
		Shared:        base,
		Publisher:     publisher,
		OutboxWorker:  outboxWorker,
		HealthHandler: healthHandler,
	}, nil
}

func (c *WorkerContainer) Close() {
	if c.DB != nil {
		db, err := c.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}

	if c.Publisher != nil {
		c.Publisher.Close()
	}
}
