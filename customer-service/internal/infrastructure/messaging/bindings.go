package messaging

import (
	"context"

	"customer-service/internal/domain/customer"

	"pizza-marketplace/shared/amqp"
)

const (
	QueueName        = "customer_queue"
	DLXName          = "customer_dlx"
	MaxRetryAttempts = 3
)

var Exchanges = map[string][]string{
	"identity.events": {
		"user.registered",
	},
	"order.events": {
		"order.address_saved",
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

func Handler(dispatcher customer.EventDispatcher) amqp.Handler {
	return func(ctx context.Context, eventName string, body []byte) error {
		return dispatcher.Dispatch(ctx, customer.EventPayload{Name: eventName, Data: body})
	}
}
