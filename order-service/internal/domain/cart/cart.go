package cart

import (
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	CustomerID   uuid.UUID  `gorm:"type:uuid;not null;unique"`
	RestaurantID uuid.UUID  `gorm:"type:uuid;not null"`
	Items        []CartItem `gorm:"foreignKey:CartID"`
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;autoCreateTime"`
	UpdatedAt    *time.Time `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (Cart) TableName() string {
	return "carts"
}

func NewCart(id, customerID, restaurantID uuid.UUID) *Cart {
	return &Cart{
		ID:           id,
		CustomerID:   customerID,
		RestaurantID: restaurantID,
	}
}

func (c *Cart) EnsureRestaurant(restaurantID uuid.UUID) error {
	if c.RestaurantID != restaurantID {
		return ErrCartRestaurantMismatch
	}

	return nil
}
