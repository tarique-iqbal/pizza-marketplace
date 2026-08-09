package restaurant

import (
	"time"

	"github.com/google/uuid"
)

type Topping struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string    `gorm:"size:64;not null"`
	Description  *string   `gorm:"size:2000"`
	IsVegetarian bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time `gorm:"type:timestamptz;autoCreateTime"`
}

func (Topping) TableName() string {
	return "toppings"
}
