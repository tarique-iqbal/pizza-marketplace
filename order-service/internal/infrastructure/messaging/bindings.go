package messaging

import (
	"context"

	"order-service/internal/domain/readmodel"

	"pizza-marketplace/shared/amqp"
)

const (
	Exchange         = "order.events"
	QueueName        = "order_queue"
	DLXName          = "order_dlx"
	MaxRetryAttempts = 3
)

var Exchanges = map[string][]string{
	"restaurant.events": {
		"restaurant.launched",
		"restaurant.updated",
		"restaurant.pizza_updated",
		"restaurant.topping_prices_updated",
	},
	"identity.events": {
		"user.registered",
	},
	"payment.events": {
		"payment.succeeded",
		"payment.failed",
	},
	"customer.events": {
		"customer.phone_updated",
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

func Handler(dispatcher readmodel.EventDispatcher) amqp.Handler {
	return func(ctx context.Context, eventName string, body []byte) error {
		return dispatcher.Dispatch(ctx, readmodel.EventPayload{Name: eventName, Data: body})
	}
}
