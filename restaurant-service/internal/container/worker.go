package container

import (
	"log/slog"

	outboxapp "restaurant-service/internal/application/outbox"
	"restaurant-service/internal/application/restaurant/inbound"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/db"
	"restaurant-service/internal/infrastructure/messaging"
	"restaurant-service/internal/infrastructure/persistence"
	"restaurant-service/internal/interfaces/http/handlers"

	"pizza-marketplace/shared/health"
)

type WorkerContainer struct {
	*Shared
	Dispatcher    restaurant.EventDispatcher
	Consumer      *messaging.RabbitMQConsumer
	Publisher     *messaging.RabbitMQPublisher
	OutboxWorker  *outboxapp.Worker
	HealthHandler *handlers.HealthHandler
}

func NewWorkerContainer(logger *slog.Logger) (*WorkerContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	restaurantRepo := persistence.NewRestaurantRepository(base.DB)
	restaurantInitiated := inbound.NewRestaurantInitiated(restaurantRepo)

	dispatcher := inbound.NewEventDispatcher()
	dispatcher.Register(messaging.Exchanges["identity.events"][0], restaurantInitiated)

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
	healthHandler := handlers.NewHealthHandler(readiness)

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
