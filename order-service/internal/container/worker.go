package container

import (
	"log/slog"

	outboxapp "order-service/internal/application/outbox"
	"order-service/internal/infrastructure/messaging"
)

type WorkerContainer struct {
	*Shared
	Publisher    *messaging.RabbitMQPublisher
	OutboxWorker *outboxapp.Worker
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

	return &WorkerContainer{
		Shared:       base,
		Publisher:    publisher,
		OutboxWorker: outboxWorker,
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
