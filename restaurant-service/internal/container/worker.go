package container

import (
	"restaurant-service/internal/application/restaurant/inbound"
	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/internal/infrastructure/messaging"
	"restaurant-service/internal/infrastructure/persistence"
)

type WorkerContainer struct {
	*Shared
	Dispatcher restaurant.EventDispatcher
	Consumer   *messaging.RabbitMQConsumer
}

func NewWorkerContainer() (*WorkerContainer, error) {
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

	return &WorkerContainer{
		Shared:     base,
		Dispatcher: dispatcher,
		Consumer:   consumer,
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
}
