package container

import (
	"log/slog"

	appcustomer "customer-service/internal/application/customer"
	"customer-service/internal/application/customer/handlers"
	outboxapp "customer-service/internal/application/outbox"
	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/messaging"
	"customer-service/internal/infrastructure/persistence"
)

type WorkerContainer struct {
	*Shared
	Dispatcher   customer.EventDispatcher
	Consumer     *messaging.RabbitMQConsumer
	Publisher    *messaging.RabbitMQPublisher
	OutboxWorker *outboxapp.Worker
}

func NewWorkerContainer(logger *slog.Logger) (*WorkerContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	customerRepo := persistence.NewCustomerRepository(base.DB)
	userRegistered := handlers.NewUserRegistered(customerRepo)

	dispatcher := appcustomer.NewEventDispatcher()
	dispatcher.Register(messaging.Exchanges["identity.events"][0], userRegistered)

	consumer, err := messaging.NewRabbitMQConsumer(base.AMQPURL)
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
		Dispatcher:   dispatcher,
		Consumer:     consumer,
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

	if c.Consumer != nil {
		c.Consumer.Close()
	}

	if c.Publisher != nil {
		c.Publisher.Close()
	}
}
