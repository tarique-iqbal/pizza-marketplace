package container

import (
	"os"

	"gorm.io/gorm"

	"order-service/internal/domain/outbox"
	"order-service/internal/infrastructure/db"
	"order-service/internal/infrastructure/persistence"
)

type Shared struct {
	AMQPURL    string
	DB         *gorm.DB
	OutboxRepo outbox.OutboxRepository
}

func NewShared() (*Shared, error) {
	postgres, err := db.NewDB()
	if err != nil {
		return nil, err
	}

	amqpURL := os.Getenv("RABBITMQ_URL")

	return &Shared{
		AMQPURL:    amqpURL,
		DB:         postgres.DB,
		OutboxRepo: persistence.NewOutboxRepository(postgres.DB),
	}, nil
}
