package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"restaurant-service/internal/domain/restaurant"
	logobs "restaurant-service/internal/infrastructure/observability/logger"
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

type messageSource interface {
	GetMessages(ctx context.Context) (<-chan amqp.Delivery, error)
	Republish(ctx context.Context, msg amqp.Delivery, retryCount int) error
}

type RabbitMQConsumer struct {
	amqpURL string
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQConsumer(amqpURL string) (*RabbitMQConsumer, error) {
	c := &RabbitMQConsumer{amqpURL: amqpURL}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RabbitMQConsumer) connect() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}

	conn, err := amqp.Dial(c.amqpURL)
	if err != nil {
		return fmt.Errorf("RabbitMQ connection failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel creation failed: %w", err)
	}

	// Declare DLX
	if err := ch.ExchangeDeclare(DLXName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("DLX declaration failed: %w", err)
	}

	// Declare main exchange
	for exchange := range Exchanges {
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("exchange declaration failed: %w", err)
		}
	}

	// Queue with DLX
	args := amqp.Table{
		"x-dead-letter-exchange":    DLXName,
		"x-dead-letter-routing-key": QueueName + ".retry",
	}
	q, err := ch.QueueDeclare(QueueName, true, false, false, false, args)
	if err != nil {
		return fmt.Errorf("queue declaration failed: %w", err)
	}

	// Bind routing keys
	for exchange, routingKeys := range Exchanges {
		for _, key := range routingKeys {
			if err := ch.QueueBind(q.Name, key, exchange, false, nil); err != nil {
				return fmt.Errorf("queue bind failed for exchange %s with key %s: %w", exchange, key, err)
			}
		}
	}

	// DLQ: messages that exhausted retries, held for inspection or replay.
	dlq, err := ch.QueueDeclare(QueueName+".dlq", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("DLQ declaration failed: %w", err)
	}

	if err := ch.QueueBind(dlq.Name, QueueName+".retry", DLXName, false, nil); err != nil {
		return fmt.Errorf("DLQ bind failed: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("QoS setup failed: %w", err)
	}

	c.conn = conn
	c.channel = ch
	return nil
}

func (c *RabbitMQConsumer) GetMessages(ctx context.Context) (<-chan amqp.Delivery, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return nil, err
	}

	msgs, err := c.channel.Consume(
		QueueName,
		"",    // let RabbitMQ generate consumer tag
		false, // manual ack
		false, // no exclusive
		false, // no local
		false, // no wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("consumer registration failed: %w", err)
	}

	return msgs, nil
}

func (c *RabbitMQConsumer) ensureConnected(ctx context.Context) error {
	if c.conn == nil || c.conn.IsClosed() || c.channel == nil || c.channel.IsClosed() {
		logobs.FromContext(ctx).Warn("reconnecting to RabbitMQ...")

		return c.connect()
	}
	return nil
}

// Run consumes until ctx is cancelled; dropped connections trigger a reconnect via
// GetMessages' ensureConnected, allowing consumption to resume without returning an error.
func Run(ctx context.Context, source messageSource, dispatcher restaurant.EventDispatcher) error {
	for {
		err := runOnce(ctx, source, dispatcher)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		logobs.FromContext(ctx).Warn("consumer connection lost, reconnecting", "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func runOnce(ctx context.Context, source messageSource, dispatcher restaurant.EventDispatcher) error {
	msgs, err := source.GetMessages(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return errors.New("message channel closed")
			}

			event := restaurant.EventPayload{
				Name: msg.RoutingKey,
				Data: msg.Body,
			}

			if err := dispatcher.Dispatch(ctx, event); err != nil {
				retryCount := getRetryCount(msg.Headers)

				if retryCount >= MaxRetryAttempts {
					logobs.FromContext(ctx).Warn(
						"message exceeded retry limit",
						"message_id", msg.MessageId,
						"retry_count", retryCount,
					)

					_ = msg.Nack(false, false) // routes to the DLX-bound DLQ for inspection

					continue
				}

				time.Sleep(time.Second * time.Duration(retryCount+1))

				if err := source.Republish(ctx, msg, retryCount+1); err != nil {
					logobs.FromContext(ctx).Warn("failed to republish for retry", "error", err)

					_ = msg.Nack(false, true) // fall back to plain requeue to avoid message loss

					continue
				}

				_ = msg.Ack(false) // superseded by the republished copy with an incremented retry count

				continue
			}

			_ = msg.Ack(false)
		}
	}
}

// Republish requeues msg via the default exchange with an incremented
// x-retry-count, which getRetryCount reads on the next delivery.
func (c *RabbitMQConsumer) Republish(
	ctx context.Context,
	msg amqp.Delivery,
	retryCount int,
) error {
	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = int32(retryCount)

	return c.channel.PublishWithContext(ctx, "", QueueName, false, false, amqp.Publishing{
		ContentType:  msg.ContentType,
		Body:         msg.Body,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
		MessageId:    msg.MessageId,
	})
}

func getRetryCount(headers amqp.Table) int {
	if val, ok := headers["x-retry-count"]; ok {
		if count, ok := val.(int32); ok {
			return int(count)
		}
	}
	return 0
}

func (c *RabbitMQConsumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
