package container

import (
	"context"
	"os"

	idxapp "search-service/internal/application/index"
	"search-service/internal/domain/index"
	"search-service/internal/infrastructure/elasticsearch"
	"search-service/internal/infrastructure/messaging"
)

type WorkerContainer struct {
	Dispatcher index.EventDispatcher
	Consumer   *messaging.RabbitMQConsumer
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

	dispatcher := idxapp.NewEventDispatcher()
	dispatcher.Register(messaging.Exchanges["restaurant.events"][0], upsertSnapshot)

	consumer, err := messaging.NewRabbitMQConsumer(amqpURL)
	if err != nil {
		return nil, err
	}

	return &WorkerContainer{
		Dispatcher: dispatcher,
		Consumer:   consumer,
	}, nil
}

func (c *WorkerContainer) Close() {
	c.Consumer.Close()
}
