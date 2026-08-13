package container

import (
	"os"

	"identity-service/internal/domain/outbox"
	"identity-service/internal/infrastructure/db"
	"identity-service/internal/infrastructure/messaging"
	"identity-service/internal/infrastructure/persistence"
)

type Shared struct {
	Postgres   *db.Postgres
	OutboxRepo outbox.OutboxRepository
	Publisher  *messaging.RabbitMQPublisher
}

func NewShared() (*Shared, error) {
	amqpURL := os.Getenv("RABBITMQ_URL")

	postgres, err := db.NewDB()
	if err != nil {
		return nil, err
	}

	outboxRepo := persistence.NewOutboxRepository(postgres.DB)

	publisher, err := messaging.NewRabbitMQPublisher(amqpURL)
	if err != nil {
		return nil, err
	}

	return &Shared{
		Postgres:   postgres,
		OutboxRepo: outboxRepo,
		Publisher:  publisher,
	}, nil
}

func (c *Shared) Close() {
	if c.Postgres.DB != nil {
		db, err := c.Postgres.DB.DB()
		if err == nil {
			_ = db.Close()
		}
	}

	if c.Publisher != nil {
		c.Publisher.Close()
	}
}
