package fixtures

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"order-service/internal/domain/readmodel"
	"order-service/tests/testutil"
)

func LoadRestaurantFixtures(t *testing.T, db *gorm.DB) []readmodel.Restaurant {
	deliveryKm := int16(10)

	restaurants := []readmodel.Restaurant{
		{
			ID:           testutil.MustNewID(),
			OwnerID:      testutil.MustNewID(),
			Name:         "Pizza Paradise",
			OwnerEmail:   "owner@pizzaparadise.de",
			Lat:          53.5511,
			Lon:          9.9937,
			DeliveryKm:   &deliveryKm,
			DeliveryFee:  decimal.NewFromFloat(2.50),
			MinimumOrder: decimal.NewFromFloat(10.00),
			Pickup:       true,
			DeliveryType: readmodel.DeliveryOwn,
			Currency:     "EUR",
			UpdatedAt:    time.Now().UTC(),
		},
		{
			ID:           testutil.MustNewID(),
			OwnerID:      testutil.MustNewID(),
			Name:         "Anatolische Küche",
			OwnerEmail:   "kontakt@anatolisch.de",
			Lat:          52.5200,
			Lon:          13.4050,
			DeliveryKm:   &deliveryKm,
			DeliveryFee:  decimal.NewFromFloat(1.99),
			MinimumOrder: decimal.NewFromFloat(15.00),
			Pickup:       false,
			DeliveryType: readmodel.DeliveryExternal,
			Currency:     "EUR",
			UpdatedAt:    time.Now().UTC(),
		},
	}

	require.NoError(t, db.Create(&restaurants).Error)

	return restaurants
}
