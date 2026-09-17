package container

import (
	"os"

	"gorm.io/gorm"

	"customer-service/internal/domain/outbox"
	"customer-service/internal/infrastructure/db"
	"customer-service/internal/infrastructure/persistence"
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

	return &Shared{
		AMQPURL:    os.Getenv("RABBITMQ_URL"),
		DB:         postgres.DB,
		OutboxRepo: persistence.NewOutboxRepository(postgres.DB),
	}, nil
}
