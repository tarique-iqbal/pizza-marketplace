package readmodel

import (
	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email     string    `gorm:"size:255;not null"`
	FirstName string    `gorm:"size:128;not null"`
	Phone     *string   `gorm:"size:32"`
}

func (Customer) TableName() string {
	return "customers"
}
