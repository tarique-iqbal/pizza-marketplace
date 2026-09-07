package order

import (
	"context"
	"time"
)

type GeocodeEntry struct {
	AddressHash string    `gorm:"type:char(64);primaryKey"`
	Lat         float64   `gorm:"type:double precision;not null;check:lat BETWEEN -90 AND 90"`
	Lon         float64   `gorm:"type:double precision;not null;check:lon BETWEEN -180 AND 180"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;autoCreateTime"`
}

func (GeocodeEntry) TableName() string {
	return "geocode"
}

type GeocodeRepository interface {
	FindByHash(ctx context.Context, hash string) (*GeocodeEntry, error)
	Create(ctx context.Context, entry GeocodeEntry) error
}
