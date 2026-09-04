package readmodel

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DeliveryType string

const (
	DeliveryOwn      DeliveryType = "own"
	DeliveryExternal DeliveryType = "external"
	DeliveryNone     DeliveryType = "none"
)

type Restaurant struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	OwnerID      uuid.UUID       `gorm:"type:uuid;not null"`
	Name         string          `gorm:"size:128;not null"`
	OwnerEmail   string          `gorm:"size:255;not null"`
	Lat          float64         `gorm:"type:double precision;not null;check:lat BETWEEN -90 AND 90"`
	Lon          float64         `gorm:"type:double precision;not null;check:lon BETWEEN -180 AND 180"`
	DeliveryKm   *int16          `gorm:"check:delivery_km BETWEEN 1 AND 25"`
	DeliveryFee  decimal.Decimal `gorm:"type:numeric(5,2);not null;default:0"`
	MinimumOrder decimal.Decimal `gorm:"type:numeric(6,2);not null;default:0"`
	Pickup       bool            `gorm:"not null;default:true"`
	DeliveryType DeliveryType    `gorm:"type:restaurant_delivery_type_enum;not null;default:'none'"`
	Currency     string          `gorm:"type:char(3);not null;default:'EUR';size:3"`
	UpdatedAt    time.Time       `gorm:"type:timestamptz;not null"`
}

func (Restaurant) TableName() string {
	return "restaurants"
}
