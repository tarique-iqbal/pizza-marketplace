package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	logobs "order-service/internal/infrastructure/observability/logger"
)

const exchangeName = "order.events"

type RabbitMQPublisher struct {
	amqpURL string

	mu      sync.Mutex
	conn    *amqp.Connection
	channel *amqp.Channel
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

	return nil
}

func (p *RabbitMQPublisher) ensureConnected(ctx context.Context) error {
	if p.conn != nil && !p.conn.IsClosed() {
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

	p.mu.Lock()
	if err := p.ensureConnected(ctx); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("ensure rabbitmq connection: %w", err)
	}
	channel := p.channel
	p.mu.Unlock()

	errCh := make(chan error, 1)

	go func() {
		err := channel.Publish(
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
		)

		// prevent goroutine leak
		select {
		case errCh <- err:
		default:
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case err := <-errCh:
		if err != nil {
			logobs.FromContext(ctx).Warn(
				"failed to publish message",
				"error", err,
				"payload", string(payload),
				"event", routingKey,
			)
		}
		return err
	}
}

func (p *RabbitMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.channel.Close()
	p.conn.Close()
}
