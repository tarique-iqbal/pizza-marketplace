package amqp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"pizza-marketplace/shared/logobs"
	"pizza-marketplace/shared/telemetry"
	"pizza-marketplace/shared/telemetry/otelamqp"
)

type Config struct {
	Queue            string
	DLX              string
	Bindings         map[string][]string
	MaxRetryAttempts int
}

type Handler func(ctx context.Context, eventName string, body []byte) error

type MessageSource interface {
	Queue() string
	MaxRetryAttempts() int
	GetMessages(ctx context.Context) (<-chan amqp091.Delivery, error)
	Republish(ctx context.Context, msg amqp091.Delivery, retryCount int) error
}

type RabbitMQConsumer struct {
	mu      sync.Mutex
	conn    *amqp091.Connection
	channel *amqp091.Channel
	amqpURL string
	cfg     Config
}

func NewRabbitMQConsumer(amqpURL string, cfg Config) (*RabbitMQConsumer, error) {
	c := &RabbitMQConsumer{amqpURL: amqpURL, cfg: cfg}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RabbitMQConsumer) Queue() string {
	return c.cfg.Queue
}

func (c *RabbitMQConsumer) MaxRetryAttempts() int {
	return c.cfg.MaxRetryAttempts
}

// connect assumes the caller already holds c.mu.
func (c *RabbitMQConsumer) connect() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}

	conn, err := amqp091.Dial(c.amqpURL)
	if err != nil {
		return fmt.Errorf("RabbitMQ connection failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel creation failed: %w", err)
	}

	if err := ch.ExchangeDeclare(c.cfg.DLX, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("DLX declaration failed: %w", err)
	}

	for exchange := range c.cfg.Bindings {
		if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("exchange declaration failed: %w", err)
		}
	}

	args := amqp091.Table{
		"x-dead-letter-exchange":    c.cfg.DLX,
		"x-dead-letter-routing-key": c.cfg.Queue + ".retry",
	}
	q, err := ch.QueueDeclare(c.cfg.Queue, true, false, false, false, args)
	if err != nil {
		return fmt.Errorf("queue declaration failed: %w", err)
	}

	for exchange, routingKeys := range c.cfg.Bindings {
		for _, key := range routingKeys {
			if err := ch.QueueBind(q.Name, key, exchange, false, nil); err != nil {
				return fmt.Errorf("queue bind failed for exchange %s with key %s: %w", exchange, key, err)
			}
		}
	}

	dlq, err := ch.QueueDeclare(c.cfg.Queue+".dlq", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("DLQ declaration failed: %w", err)
	}

	if err := ch.QueueBind(dlq.Name, c.cfg.Queue+".retry", c.cfg.DLX, false, nil); err != nil {
		return fmt.Errorf("DLQ bind failed: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("QoS setup failed: %w", err)
	}

	c.conn = conn
	c.channel = ch
	return nil
}

func (c *RabbitMQConsumer) GetMessages(ctx context.Context) (<-chan amqp091.Delivery, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return nil, err
	}

	c.mu.Lock()
	channel := c.channel
	c.mu.Unlock()

	msgs, err := channel.Consume(c.cfg.Queue, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consumer registration failed: %w", err)
	}

	return msgs, nil
}

func (c *RabbitMQConsumer) ensureConnected(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.conn.IsClosed() || c.channel == nil || c.channel.IsClosed() {
		logobs.FromContext(ctx).WarnContext(ctx, "reconnecting to RabbitMQ...")
		return c.connect()
	}
	return nil
}

// Run consumes until ctx is cancelled; dropped connections trigger a reconnect via
// GetMessages' ensureConnected, allowing consumption to resume without returning an error.
func Run(ctx context.Context, source MessageSource, handler Handler) error {
	for {
		err := runOnce(ctx, source, handler)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		logobs.FromContext(ctx).WarnContext(ctx, "consumer connection lost, reconnecting", "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func runOnce(ctx context.Context, source MessageSource, handler Handler) error {
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

			eventName := getEventName(msg)

			msgCtx, span := otelamqp.StartConsume(ctx, source.Queue(), getExchange(msg), eventName, msg)
			msgCtx = logobs.WithContext(msgCtx, logobs.FromContext(ctx).With(
				"event_name", eventName,
				"message_id", msg.MessageId,
			))

			err := handler(msgCtx, eventName, msg.Body)
			telemetry.EndSpan(span, err)

			if err != nil {
				retryCount := getRetryCount(msg.Headers)

				if retryCount >= source.MaxRetryAttempts() {
					logobs.FromContext(msgCtx).WarnContext(msgCtx,
						"message exceeded retry limit",
						"retry_count", retryCount,
					)

					_ = msg.Nack(false, false)

					continue
				}

				time.Sleep(time.Second * time.Duration(retryCount+1))

				if err := source.Republish(ctx, msg, retryCount+1); err != nil {
					logobs.FromContext(msgCtx).WarnContext(msgCtx,
						"failed to republish for retry",
						"error", err,
					)

					_ = msg.Nack(false, true)

					continue
				}

				_ = msg.Ack(false)

				continue
			}

			_ = msg.Ack(false)
		}
	}
}

// Republish requeues msg via the default exchange with an incremented
// x-retry-count, which getRetryCount reads on the next delivery. The default exchange
// routes by queue name, so the original routing key and exchange are kept in x-original-*.
func (c *RabbitMQConsumer) Republish(ctx context.Context, msg amqp091.Delivery, retryCount int) error {
	headers := amqp091.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = int32(retryCount)
	if _, ok := headers["x-original-routing-key"]; !ok {
		headers["x-original-routing-key"] = msg.RoutingKey
	}
	if _, ok := headers["x-original-exchange"]; !ok {
		headers["x-original-exchange"] = msg.Exchange
	}

	c.mu.Lock()
	channel := c.channel
	c.mu.Unlock()

	return channel.PublishWithContext(ctx, "", c.cfg.Queue, false, false, amqp091.Publishing{
		ContentType:  msg.ContentType,
		Body:         msg.Body,
		Headers:      headers,
		DeliveryMode: amqp091.Persistent,
		MessageId:    msg.MessageId,
	})
}

func getEventName(msg amqp091.Delivery) string {
	if name, ok := msg.Headers["x-original-routing-key"].(string); ok && name != "" {
		return name
	}
	return msg.RoutingKey
}

func getExchange(msg amqp091.Delivery) string {
	if exchange, ok := msg.Headers["x-original-exchange"].(string); ok {
		return exchange
	}
	return msg.Exchange
}

func getRetryCount(headers amqp091.Table) int {
	if val, ok := headers["x-retry-count"]; ok {
		if count, ok := val.(int32); ok {
			return int(count)
		}
	}
	return 0
}

func (c *RabbitMQConsumer) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *RabbitMQConsumer) Ping(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.conn.IsClosed() {
		return amqp091.ErrClosed
	}

	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return nil
}
