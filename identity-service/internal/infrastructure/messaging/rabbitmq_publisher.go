package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	logobs "identity-service/internal/infrastructure/observability/logger"
)

const exchangeName = "identity.events"

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

	// amqp091-go broadcasts every incoming confirmation to every listener ever registered
	// via NotifyPublish on this channel, and offers no way to remove one. p.confirms is
	// therefore registered exactly once, for the channel's whole lifetime: Publish (below)
	// holds p.mu for the full round trip, guaranteeing at most one confirmation is ever
	// outstanding against this listener at a time. An unread confirmation sitting in its
	// cap-1 buffer would otherwise block the next broadcast into it, which blocks the
	// connection's single frame-reading goroutine, which blocks every future publish.
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

	// publishLocked holds p.mu for the full publish-and-await-confirm round trip,
	// independent of ctx, so at most one real publish is ever outstanding against
	// p.confirms at a time (see connect()). The caller stays bounded by ctx and returns
	// promptly below on cancellation; only publishLocked's own completion releases the
	// lock, so a caller giving up early never lets a second real publish start before
	// this one's confirmation has actually been read.
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

// publishLocked always runs under p.mu to completion, deliberately ignoring ctx
// cancellation for its own blocking calls. See the comment in Publish for why.
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
