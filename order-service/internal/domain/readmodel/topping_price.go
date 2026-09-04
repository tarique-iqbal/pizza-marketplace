package readmodel

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ToppingPrice struct {
	RestaurantID uuid.UUID       `gorm:"type:uuid;primaryKey"`
	ToppingID    uuid.UUID       `gorm:"type:uuid;primaryKey"`
	Name         string          `gorm:"size:255;not null"`
	ExtraPrice   decimal.Decimal `gorm:"type:numeric(6,2);not null"`
	UpdatedAt    time.Time       `gorm:"type:timestamptz;not null"`
}

func (ToppingPrice) TableName() string {
	return "topping_prices"
}
