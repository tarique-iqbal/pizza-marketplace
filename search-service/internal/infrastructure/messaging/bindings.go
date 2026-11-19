package messaging

import (
	"context"

	"search-service/internal/domain/index"

	"pizza-marketplace/shared/amqp"
)

const (
	QueueName        = "search_queue"
	DLXName          = "search_dlx"
	MaxRetryAttempts = 3
)

var Exchanges = map[string][]string{
	"restaurant.events": {
		"restaurant.launched",
		"restaurant.updated",
		"restaurant.pizza_updated",
		"restaurant.topping_prices_updated",
	},
}

func ConsumerConfig() amqp.Config {
	return amqp.Config{
		Queue:            QueueName,
		DLX:              DLXName,
		Bindings:         Exchanges,
		MaxRetryAttempts: MaxRetryAttempts,
	}
}

func Handler(dispatcher index.EventDispatcher) amqp.Handler {
	return func(ctx context.Context, eventName string, body []byte) error {
		return dispatcher.Dispatch(ctx, index.EventPayload{Name: eventName, Data: body})
	}
}
