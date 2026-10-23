package container

import (
	"log/slog"

	appcustomer "customer-service/internal/application/customer"
	"customer-service/internal/application/customer/handlers"
	outboxapp "customer-service/internal/application/outbox"
	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/db"
	"customer-service/internal/infrastructure/messaging"
	"customer-service/internal/infrastructure/persistence"
	httphandlers "customer-service/internal/interfaces/http/handlers"

	"pizza-marketplace/shared/health"
)

type WorkerContainer struct {
	*Shared
	Dispatcher    customer.EventDispatcher
	Consumer      *messaging.RabbitMQConsumer
	Publisher     *messaging.RabbitMQPublisher
	OutboxWorker  *outboxapp.Worker
	HealthHandler *httphandlers.HealthHandler
}

func NewWorkerContainer(logger *slog.Logger) (*WorkerContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	customerRepo := persistence.NewCustomerRepository(base.DB)
	addressRepo := persistence.NewAddressRepository(base.DB)
	userRegistered := handlers.NewUserRegistered(customerRepo)
	addressSaved := handlers.NewAddressSaved(addressRepo)

	dispatcher := appcustomer.NewEventDispatcher()
	dispatcher.Register("user.registered", userRegistered)
	dispatcher.Register("order.address_saved", addressSaved)

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

	readiness := health.NewReadiness(
		health.Check{Name: "postgres", Pinger: &db.Postgres{DB: base.DB}},
		health.Check{Name: "rabbitmq", Pinger: publisher},
	)
	healthHandler := httphandlers.NewHealthHandler(readiness)

	return &WorkerContainer{
		Shared:        base,
		Dispatcher:    dispatcher,
		Consumer:      consumer,
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

	if c.Consumer != nil {
		c.Consumer.Close()
	}

	if c.Publisher != nil {
		c.Publisher.Close()
	}
}
