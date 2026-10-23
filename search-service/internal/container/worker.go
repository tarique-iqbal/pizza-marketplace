package container

import (
	"context"
	"os"

	idxapp "search-service/internal/application/index"
	"search-service/internal/domain/index"
	"search-service/internal/infrastructure/elasticsearch"
	"search-service/internal/infrastructure/messaging"
	"search-service/internal/interfaces/http/handlers"

	"pizza-marketplace/shared/health"
)

type WorkerContainer struct {
	Dispatcher    index.EventDispatcher
	Consumer      *messaging.RabbitMQConsumer
	HealthHandler *handlers.HealthHandler
}

func NewWorkerContainer() (*WorkerContainer, error) {
	amqpURL := os.Getenv("RABBITMQ_URL")
	esURL := os.Getenv("ELASTICSEARCH_URL")

	es, err := elasticsearch.NewClient(esURL)
	if err != nil {
		return nil, err
	}

	if err := elasticsearch.EnsureIndex(context.Background(), es); err != nil {
		return nil, err
	}

	searchRepo := elasticsearch.NewSearchRepository(es)
	upsertSnapshot := idxapp.NewUpsertSnapshot(searchRepo)
	updateRestaurantFields := idxapp.NewUpdateRestaurantFields(searchRepo)
	syncPizza := idxapp.NewSyncPizza(searchRepo)
	syncToppingPrices := idxapp.NewSyncToppingPrices(searchRepo)

	dispatcher := idxapp.NewEventDispatcher()
	dispatcher.Register("restaurant.launched", upsertSnapshot)
	dispatcher.Register("restaurant.updated", updateRestaurantFields)
	dispatcher.Register("restaurant.pizza_updated", syncPizza)
	dispatcher.Register("restaurant.topping_prices_updated", syncToppingPrices)

	consumer, err := messaging.NewRabbitMQConsumer(amqpURL)
	if err != nil {
		return nil, err
	}

	readiness := health.NewReadiness(
		health.Check{Name: "elasticsearch", Pinger: elasticsearch.Pinger{ES: es}},
		health.Check{Name: "rabbitmq", Pinger: consumer},
	)
	healthHandler := handlers.NewHealthHandler(readiness)

	return &WorkerContainer{
		Dispatcher:    dispatcher,
		Consumer:      consumer,
		HealthHandler: healthHandler,
	}, nil
}

func (c *WorkerContainer) Close() {
	c.Consumer.Close()
}
