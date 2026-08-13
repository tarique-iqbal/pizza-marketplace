package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"identity-service/internal/shared/event"
)

const exchangeName = "identity.events"

type RabbitMQPublisher struct {
	amqpURL string

	mu      sync.Mutex
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(amqpURL string) *RabbitMQPublisher {
	p := &RabbitMQPublisher{amqpURL: amqpURL}

	if err := p.connect(); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	return p
}

func (p *RabbitMQPublisher) connect() error {
	conn, err := amqp.Dial(p.amqpURL)
	if err != nil {
		return fmt.Errorf("connect to rabbitmq: %w", err)
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

	return nil
}

func (p *RabbitMQPublisher) ensureConnected() error {
	if p.conn != nil && !p.conn.IsClosed() {
		return nil
	}

	log.Println("reconnecting to RabbitMQ...")

	return p.connect()
}

func (p *RabbitMQPublisher) PublishEvent(ctx context.Context, evt event.Event) error {
	body, err := json.Marshal(evt)
	if err != nil {
		log.Printf("failed to marshal event %s: %v", evt.GetEventName(), err)
		return err
	}

	return p.publish(ctx, evt.GetEventName(), body)
}

func (p *RabbitMQPublisher) PublishRaw(
	ctx context.Context,
	routingKey string,
	body []byte,
) error {
	return p.publish(ctx, routingKey, body)
}

func (p *RabbitMQPublisher) publish(
	ctx context.Context,
	routingKey string,
	body []byte,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Publish and confirm-wait are serialized on p.mu: confirms aren't tagged to a
	// specific call, so overlapping publishes could read back the wrong confirmation.
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureConnected(); err != nil {
		return fmt.Errorf("ensure rabbitmq connection: %w", err)
	}

	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	errCh := make(chan error, 1)

	go func() {
		err := p.channel.Publish(
			exchangeName,
			routingKey,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				Body:         body,
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
			log.Printf(
				"Failed to publish message: %v body: %s event: %s",
				err, body, routingKey,
			)
			return err
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()

	case confirm, ok := <-confirms:
		if !ok {
			return fmt.Errorf("publisher confirm channel closed before ack for event %s", routingKey)
		}
		if !confirm.Ack {
			return fmt.Errorf("broker nacked publish for event %s", routingKey)
		}
		return nil
	}
}

func (p *RabbitMQPublisher) Ping(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil || p.conn.IsClosed() {
		return amqp.ErrClosed
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return nil
}

func (p *RabbitMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.channel.Close()
	p.conn.Close()
}
