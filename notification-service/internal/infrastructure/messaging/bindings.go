package messaging

import (
	"context"

	"notification-service/internal/domain/notification"

	"pizza-marketplace/shared/amqp"
)

const (
	QueueName        = "email_queue"
	DLXName          = "email_dlx"
	MaxRetryAttempts = 3
)

var Exchanges = map[string][]string{
	"identity.events": {
		"email.verification_created",
		"user.registered",
	},
	"restaurant.events": {
		"restaurant.ready_for_review",
		"restaurant.approved",
	},
	"order.events": {
		"order.confirmed",
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

func Handler(dispatcher notification.EventDispatcher) amqp.Handler {
	return func(ctx context.Context, eventName string, body []byte) error {
		return dispatcher.Dispatch(ctx, notification.EventPayload{Name: eventName, Data: body})
	}
}
