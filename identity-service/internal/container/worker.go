package container

import (
	"log/slog"
	"os"

	"identity-service/internal/application/outbox"
	"identity-service/internal/infrastructure/messaging"
	"identity-service/internal/interfaces/http"

	"pizza-marketplace/shared/amqp"
	"pizza-marketplace/shared/health"
)

type WorkerContainer struct {
	*Shared
	Publisher     *amqp.RabbitMQPublisher
	Worker        *outbox.Worker
	HealthHandler *http.HealthHandler
}

func NewWorkerContainer(logger *slog.Logger) (*WorkerContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	publisher, err := amqp.NewRabbitMQPublisher(os.Getenv("RABBITMQ_URL"), messaging.Exchange)
	if err != nil {
		return nil, err
	}

	relayer := outbox.NewRelay(publisher)
	worker := outbox.NewWorker(base.OutboxRepo, relayer, outbox.DefaultConfig(), logger)

	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: base.Postgres},
		health.Check{Name: "rabbitmq-publisher", Pinger: publisher},
	)
	healthHandler := http.NewHealthHandler(readiness)

	return &WorkerContainer{
		Shared:        base,
		Publisher:     publisher,
		Worker:        worker,
		HealthHandler: healthHandler,
	}, nil
}

func (c *WorkerContainer) Close() {
	c.Shared.Close()

	if c.Publisher != nil {
		c.Publisher.Close()
	}
}
