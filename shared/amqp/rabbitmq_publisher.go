package amqp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"pizza-marketplace/shared/logobs"
	"pizza-marketplace/shared/telemetry"
	"pizza-marketplace/shared/telemetry/otelamqp"
)

type RabbitMQPublisher struct {
	amqpURL  string
	exchange string

	mu       sync.Mutex
	conn     *amqp091.Connection
	channel  *amqp091.Channel
	confirms chan amqp091.Confirmation
}

func NewRabbitMQPublisher(amqpURL, exchange string) (*RabbitMQPublisher, error) {
	p := &RabbitMQPublisher{amqpURL: amqpURL, exchange: exchange}

	if err := p.connect(); err != nil {
		return nil, err
	}

	return p, nil
}

// connect assumes the caller already holds p.mu (or, from the constructor, that nothing else can).
func (p *RabbitMQPublisher) connect() error {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}

	conn, err := amqp091.Dial(p.amqpURL)
	if err != nil {
		return fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return fmt.Errorf("enable publisher confirms: %w", err)
	}

	if err := ch.ExchangeDeclare(p.exchange, "topic", true, false, false, false, nil); err != nil {
		_ = conn.Close()
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
	p.confirms = ch.NotifyPublish(make(chan amqp091.Confirmation, 1))

	return nil
}

func (p *RabbitMQPublisher) ensureConnected(ctx context.Context) error {
	if p.conn != nil && !p.conn.IsClosed() && p.channel != nil && !p.channel.IsClosed() {
		return nil
	}

	logobs.FromContext(ctx).WarnContext(ctx, "reconnecting to RabbitMQ...")

	return p.connect()
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, messageID, routingKey string, payload []byte) error {
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

		resultCh <- p.publishLocked(ctx, messageID, routingKey, payload)
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
func (p *RabbitMQPublisher) publishLocked(
	ctx context.Context,
	messageID string,
	routingKey string,
	payload []byte,
) (err error) {
	headers := amqp091.Table{"x-event-name": routingKey}

	ctx, span := otelamqp.StartPublish(ctx, p.exchange, routingKey, headers)
	defer func() { telemetry.EndSpan(span, err) }()

	if err := p.ensureConnected(ctx); err != nil {
		return fmt.Errorf("ensure rabbitmq connection: %w", err)
	}

	if err := p.channel.Publish(
		p.exchange,
		routingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp091.Persistent,
			MessageId:    messageID,
			Timestamp:    time.Now().UTC(),
			Type:         routingKey,
			Headers:      headers,
		},
	); err != nil {
		logobs.FromContext(ctx).WarnContext(ctx,
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

	if err := p.ensureConnected(ctx); err != nil {
		return err
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
