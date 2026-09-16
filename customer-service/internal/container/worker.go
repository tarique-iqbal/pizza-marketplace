package container

import (
	appcustomer "customer-service/internal/application/customer"
	"customer-service/internal/application/customer/handlers"
	"customer-service/internal/domain/customer"
	"customer-service/internal/infrastructure/messaging"
	"customer-service/internal/infrastructure/persistence"
)

type WorkerContainer struct {
	*Shared
	Dispatcher customer.EventDispatcher
	Consumer   *messaging.RabbitMQConsumer
}

func NewWorkerContainer() (*WorkerContainer, error) {
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
