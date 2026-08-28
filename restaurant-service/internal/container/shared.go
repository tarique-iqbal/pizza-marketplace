package container

import (
	"os"

	"gorm.io/gorm"

	"restaurant-service/internal/domain/outbox"
	"restaurant-service/internal/infrastructure/db"
	"restaurant-service/internal/infrastructure/persistence"
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
