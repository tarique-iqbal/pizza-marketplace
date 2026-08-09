package restaurant

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PizzaPrice struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey"`
	PizzaID   uuid.UUID       `gorm:"column:pizza_id;type:uuid;not null"`
	SizeID    uuid.UUID       `gorm:"column:size_id;type:uuid;not null"`
	Price     decimal.Decimal `gorm:"type:numeric(6,2);not null"`
	IsActive  bool            `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time       `gorm:"type:timestamptz;autoCreateTime"`
	UpdatedAt *time.Time      `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (PizzaPrice) TableName() string {
	return "pizza_prices"
}

func NewPizzaPrice(pizzaID uuid.UUID, sizeID uuid.UUID, price decimal.Decimal) (*PizzaPrice, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &PizzaPrice{
		ID:       id,
		PizzaID:  pizzaID,
		SizeID:   sizeID,
		Price:    price,
		IsActive: true,
	}, nil
}
