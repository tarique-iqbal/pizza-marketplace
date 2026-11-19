package messaging

import (
	"context"

	"restaurant-service/internal/domain/restaurant"

	"pizza-marketplace/shared/amqp"
)

const (
	QueueName        = "restaurant_queue"
	DLXName          = "restaurant_dlx"
	MaxRetryAttempts = 3
)

var Exchanges = map[string][]string{
	"identity.events": {
		"restaurant.initiated",
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

func Handler(dispatcher restaurant.EventDispatcher) amqp.Handler {
	return func(ctx context.Context, eventName string, body []byte) error {
		return dispatcher.Dispatch(ctx, restaurant.EventPayload{Name: eventName, Data: body})
	}
}
