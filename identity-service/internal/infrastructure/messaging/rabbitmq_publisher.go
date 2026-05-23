package messaging

import (
	"context"
	"encoding/json"
	"identity-service/internal/shared/event"
	"log"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "identity.events"

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(amqpURL string) *RabbitMQPublisher {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
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
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	return &RabbitMQPublisher{conn: conn, channel: ch}
}

func (p *RabbitMQPublisher) PublishEvent(ctx context.Context, event event.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.publish(ctx, event.GetEventName(), body)
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
		}
		return err
	}
}

func (p *RabbitMQPublisher) Ping(ctx context.Context) error {
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
	p.channel.Close()
	p.conn.Close()
}
