package readmodel

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PizzaPrice struct {
	PizzaID    uuid.UUID       `gorm:"type:uuid;primaryKey"`
	SizeID     uuid.UUID       `gorm:"type:uuid;primaryKey"`
	DiameterCm int16           `gorm:"not null"`
	Price      decimal.Decimal `gorm:"type:numeric(6,2);not null"`
	IsActive   bool            `gorm:"not null;default:true"`
	UpdatedAt  time.Time       `gorm:"type:timestamptz;not null"`
}

func (PizzaPrice) TableName() string {
	return "pizza_prices"
}
