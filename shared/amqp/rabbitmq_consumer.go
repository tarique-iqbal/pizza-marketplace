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

const DefaultRetryDelay = 5 * time.Second

type Config struct {
	Queue            string
	DLX              string
	Bindings         map[string][]string
	MaxRetryAttempts int
	RetryDelay       time.Duration
}

type Handler func(ctx context.Context, eventName string, body []byte) error

type MessageSource interface {
	Queue() string
	MaxRetryAttempts() int
	GetMessages(ctx context.Context) (<-chan amqp091.Delivery, error)
	DeadLetter(ctx context.Context, msg amqp091.Delivery) error
}

type RabbitMQConsumer struct {
	mu      sync.Mutex
	conn    *amqp091.Connection
	channel *amqp091.Channel
	amqpURL string
	cfg     Config
}

func NewRabbitMQConsumer(amqpURL string, cfg Config) (*RabbitMQConsumer, error) {
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = DefaultRetryDelay
	}

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

func (c *RabbitMQConsumer) waitQueue() string {
	return c.cfg.Queue + ".wait"
}

func (c *RabbitMQConsumer) dlq() string {
	return c.cfg.Queue + ".dlq"
}

func (c *RabbitMQConsumer) retryKey() string {
	return c.cfg.Queue + ".retry"
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

	if err := c.declareTopology(ch); err != nil {
		return err
	}

	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("confirm mode setup failed: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("QoS setup failed: %w", err)
	}

	c.conn = conn
	c.channel = ch
	return nil
}

func (c *RabbitMQConsumer) declareTopology(ch *amqp091.Channel) error {
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
		"x-dead-letter-routing-key": c.retryKey(),
	}
	if _, err := ch.QueueDeclare(c.cfg.Queue, true, false, false, false, args); err != nil {
		return fmt.Errorf("queue declaration failed: %w", err)
	}

	for exchange, routingKeys := range c.cfg.Bindings {
		for _, key := range routingKeys {
			if err := ch.QueueBind(c.cfg.Queue, key, exchange, false, nil); err != nil {
				return fmt.Errorf("queue bind failed for exchange %s with key %s: %w", exchange, key, err)
			}
		}
	}

	waitArgs := amqp091.Table{
		"x-message-ttl":             c.cfg.RetryDelay.Milliseconds(),
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": c.cfg.Queue,
	}
	if _, err := ch.QueueDeclare(c.waitQueue(), true, false, false, false, waitArgs); err != nil {
		return fmt.Errorf("wait queue declaration failed: %w", err)
	}

	if err := ch.QueueBind(c.waitQueue(), c.retryKey(), c.cfg.DLX, false, nil); err != nil {
		return fmt.Errorf("wait queue bind failed: %w", err)
	}

	if _, err := ch.QueueDeclare(c.dlq(), true, false, false, false, nil); err != nil {
		return fmt.Errorf("DLQ declaration failed: %w", err)
	}

	if err := ch.QueueUnbind(c.dlq(), c.retryKey(), c.cfg.DLX, nil); err != nil {
		return fmt.Errorf("DLQ unbind of retry key failed: %w", err)
	}

	if err := ch.QueueBind(c.dlq(), c.dlq(), c.cfg.DLX, false, nil); err != nil {
		return fmt.Errorf("DLQ bind failed: %w", err)
	}

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

			handle(ctx, source, handler, msg)
		}
	}
}

func handle(ctx context.Context, source MessageSource, handler Handler, msg amqp091.Delivery) {
	death := firstRejection(msg.Headers, source.Queue())
	eventName, exchange := msg.RoutingKey, msg.Exchange
	if death != nil && death.routingKey != "" {
		eventName, exchange = death.routingKey, death.exchange
	}

	msgCtx, span := otelamqp.StartConsume(ctx, source.Queue(), exchange, eventName, msg)
	msgCtx = logobs.WithContext(msgCtx, logobs.FromContext(ctx).With(
		"event_name", eventName,
		"message_id", msg.MessageId,
	))

	err := handler(msgCtx, eventName, msg.Body)
	telemetry.EndSpan(span, err)

	if err == nil {
		_ = msg.Ack(false)
		return
	}

	retries := 0
	if death != nil {
		retries = death.count
	}

	if retries < source.MaxRetryAttempts() {
		_ = msg.Nack(false, false)
		return
	}

	logobs.FromContext(msgCtx).WarnContext(msgCtx, "message exceeded retry limit", "retry_count", retries)

	if err := source.DeadLetter(ctx, msg); err != nil {
		logobs.FromContext(msgCtx).WarnContext(msgCtx, "failed to move message to DLQ", "error", err)
		_ = msg.Nack(false, false)
		return
	}

	_ = msg.Ack(false)
}

// DeadLetter publishes msg to the DLQ and returns once the broker has confirmed it, so the
// caller only acks the original after the copy is safe.
func (c *RabbitMQConsumer) DeadLetter(ctx context.Context, msg amqp091.Delivery) error {
	c.mu.Lock()
	channel := c.channel
	c.mu.Unlock()

	confirm, err := channel.PublishWithDeferredConfirmWithContext(ctx, c.cfg.DLX, c.dlq(), false, false,
		amqp091.Publishing{
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			Headers:      msg.Headers,
			DeliveryMode: amqp091.Persistent,
			MessageId:    msg.MessageId,
		})
	if err != nil {
		return fmt.Errorf("DLQ publish failed: %w", err)
	}

	acked, err := confirm.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("DLQ publish confirm failed: %w", err)
	}
	if !acked {
		return errors.New("DLQ publish nacked by broker")
	}
	return nil
}

type rejection struct {
	count      int
	exchange   string
	routingKey string
}

// firstRejection reads the x-death entry the broker keeps for rejections from queue. It holds
// the message's original exchange and routing key on every pass, and how often it was rejected.
func firstRejection(headers amqp091.Table, queue string) *rejection {
	deaths, ok := headers["x-death"].([]interface{})
	if !ok {
		return nil
	}

	for _, d := range deaths {
		entry, ok := d.(amqp091.Table)
		if !ok || entry["queue"] != queue || entry["reason"] != "rejected" {
			continue
		}

		r := &rejection{count: toInt(entry["count"])}
		r.exchange, _ = entry["exchange"].(string)
		if keys, ok := entry["routing-keys"].([]interface{}); ok && len(keys) > 0 {
			r.routingKey, _ = keys[0].(string)
		}
		return r
	}
	return nil
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int32:
		return int(n)
	case int:
		return n
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
