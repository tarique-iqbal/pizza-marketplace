package topping

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ToppingPrice struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	RestaurantID uuid.UUID       `gorm:"column:restaurant_id;type:uuid;not null"`
	ToppingID    uuid.UUID       `gorm:"column:topping_id;type:uuid;not null"`
	ExtraPrice   decimal.Decimal `gorm:"column:extra_price;type:numeric(6,2);not null"`
	CreatedAt    time.Time       `gorm:"type:timestamptz;autoCreateTime"`
	UpdatedAt    *time.Time      `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (ToppingPrice) TableName() string {
	return "topping_prices"
}

func NewToppingPrice(
	restaurantID uuid.UUID,
	toppingID uuid.UUID,
	extraPrice decimal.Decimal,
) (*ToppingPrice, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &ToppingPrice{
		ID:           id,
		RestaurantID: restaurantID,
		ToppingID:    toppingID,
		ExtraPrice:   extraPrice,
	}, nil
}
