package messaging

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"email-service/internal/domain/email"

	"github.com/rabbitmq/amqp091-go"
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
	},
}

type RabbitMQConsumer struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	amqpURL string
}

func NewRabbitMQConsumer(amqpURL string) (*RabbitMQConsumer, error) {
	c := &RabbitMQConsumer{amqpURL: amqpURL}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RabbitMQConsumer) connect() error {
	conn, err := amqp091.Dial(c.amqpURL)
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
	args := amqp091.Table{
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

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("QoS setup failed: %w", err)
	}

	c.conn = conn
	c.channel = ch
	return nil
}

func (c *RabbitMQConsumer) GetMessages() (<-chan amqp091.Delivery, error) {
	if err := c.ensureConnected(); err != nil {
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

func (c *RabbitMQConsumer) ensureConnected() error {
	if c.conn.IsClosed() {
		log.Println("Reconnecting to RabbitMQ...")
		return c.connect()
	}
	return nil
}

func (c *RabbitMQConsumer) Run(ctx context.Context, dispatcher email.EventDispatcher) error {
	msgs, err := c.GetMessages()
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return errors.New("message channel closed")
			}

			event := email.EventPayload{
				Name: msg.RoutingKey,
				Data: msg.Body,
			}

			if err := dispatcher.Dispatch(event); err != nil {
				retryCount := getRetryCount(msg.Headers)
				if retryCount >= MaxRetryAttempts {
					log.Printf("Message %s exceeded retry limit", msg.MessageId)
					_ = msg.Nack(false, false) // discard
					continue
				}

				time.Sleep(time.Second * time.Duration(retryCount+1))
				_ = msg.Nack(false, true) // requeue
				continue
			}

			_ = msg.Ack(false)
		}
	}
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
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
