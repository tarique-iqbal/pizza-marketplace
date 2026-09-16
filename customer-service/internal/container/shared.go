package container

import (
	"os"

	"gorm.io/gorm"

	"customer-service/internal/infrastructure/db"
)

type Shared struct {
	AMQPURL string
	DB      *gorm.DB
}

func NewShared() (*Shared, error) {
	postgres, err := db.NewDB()
	if err != nil {
		return nil, err
	}

	return &Shared{
		AMQPURL: os.Getenv("RABBITMQ_URL"),
		DB:      postgres.DB,
	}, nil
}
