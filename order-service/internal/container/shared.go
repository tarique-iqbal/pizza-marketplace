package container

import (
	"gorm.io/gorm"

	"order-service/internal/infrastructure/db"
)

type Shared struct {
	DB *gorm.DB
}

func NewShared() (*Shared, error) {
	postgres, err := db.NewDB()
	if err != nil {
		return nil, err
	}

	return &Shared{
		DB: postgres.DB,
	}, nil
}
