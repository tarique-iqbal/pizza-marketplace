package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	logobs "restaurant-service/internal/infrastructure/observability/logger"
)

const exchangeName = "restaurant.events"

type RabbitMQPublisher struct {
	amqpURL string

	mu       sync.Mutex
	conn     *amqp.Connection
	channel  *amqp.Channel
	confirms chan amqp.Confirmation
}

func NewRabbitMQPublisher(amqpURL string) (*RabbitMQPublisher, error) {
	p := &RabbitMQPublisher{amqpURL: amqpURL}

	if err := p.connect(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *RabbitMQPublisher) connect() error {
	conn, err := amqp.Dial(p.amqpURL)
	if err != nil {
		return fmt.Errorf("rabbitmq connect: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}

	err = ch.ExchangeDeclare(
		exchangeName, // Exchange name
		"topic",      // Exchange type
		true,         // Durable
		false,        // Auto-delete
		false,        // Internal
		false,        // No-wait
		nil,          // Arguments
	)
	if err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	p.conn = conn
	p.channel = ch
	p.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	return nil
}

func (p *RabbitMQPublisher) ensureConnected(ctx context.Context) error {
	if p.conn != nil && !p.conn.IsClosed() && p.channel != nil && !p.channel.IsClosed() {
		return nil
	}

	logobs.FromContext(ctx).Warn("reconnecting to RabbitMQ...")

	return p.connect()
}

func (p *RabbitMQPublisher) Publish(
	ctx context.Context,
	routingKey string,
	payload []byte,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	resultCh := make(chan error, 1)

	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		resultCh <- p.publishLocked(ctx, routingKey, payload)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-resultCh:
		return err
	}
}

func (p *RabbitMQPublisher) publishLocked(ctx context.Context, routingKey string, payload []byte) error {
	if err := p.ensureConnected(ctx); err != nil {
		return fmt.Errorf("ensure rabbitmq connection: %w", err)
	}

	if err := p.channel.Publish(
		exchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp.Persistent,
			MessageId:    uuid.NewString(),
			Timestamp:    time.Now().UTC(),
			Type:         routingKey,
			Headers: amqp.Table{
				"x-event-name": routingKey,
			},
		},
	); err != nil {
		logobs.FromContext(ctx).Warn(
			"failed to publish message",
			"error", err,
			"payload", string(payload),
			"event", routingKey,
		)
		return err
	}

	confirm, ok := <-p.confirms
	if !ok {
		return fmt.Errorf("publisher confirm channel closed before ack for event %s", routingKey)
	}
	if !confirm.Ack {
		return fmt.Errorf("broker nacked publish for event %s", routingKey)
	}
	return nil
}

func (p *RabbitMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.channel.Close()
	p.conn.Close()
}
