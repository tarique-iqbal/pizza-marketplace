package container

import (
	"log/slog"

	outboxapp "order-service/internal/application/outbox"
	appreadmodel "order-service/internal/application/readmodel"
	"order-service/internal/domain/readmodel"
	"order-service/internal/infrastructure/messaging"
	"order-service/internal/infrastructure/persistence"
)

type WorkerContainer struct {
	*Shared
	Dispatcher   readmodel.EventDispatcher
	Consumer     *messaging.RabbitMQConsumer
	Publisher    *messaging.RabbitMQPublisher
	OutboxWorker *outboxapp.Worker
}

func NewWorkerContainer(logger *slog.Logger) (*WorkerContainer, error) {
	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	restaurantRepo := persistence.NewRestaurantRepository(base.DB)
	pizzaRepo := persistence.NewPizzaRepository(base.DB)
	pizzaPriceRepo := persistence.NewPizzaPriceRepository(base.DB)
	toppingPriceRepo := persistence.NewToppingPriceRepository(base.DB)
	customerRepo := persistence.NewCustomerRepository(base.DB)

	upsertRestaurant := appreadmodel.NewUpsertRestaurant(restaurantRepo, pizzaRepo, pizzaPriceRepo, toppingPriceRepo)
	updateRestaurant := appreadmodel.NewUpdateRestaurant(restaurantRepo)
	syncPizza := appreadmodel.NewSyncPizza(pizzaRepo, pizzaPriceRepo)
	syncToppingPrices := appreadmodel.NewSyncToppingPrices(toppingPriceRepo)
	upsertCustomer := appreadmodel.NewUpsertCustomer(customerRepo)

	dispatcher := appreadmodel.NewEventDispatcher()
	dispatcher.Register("restaurant.launched", upsertRestaurant)
	dispatcher.Register("restaurant.updated", updateRestaurant)
	dispatcher.Register("restaurant.pizza_updated", syncPizza)
	dispatcher.Register("restaurant.topping_prices_updated", syncToppingPrices)
	dispatcher.Register("user.registered", upsertCustomer)

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
