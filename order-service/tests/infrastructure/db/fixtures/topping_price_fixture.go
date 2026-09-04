package fixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"order-service/internal/domain/readmodel"
	"order-service/tests/testutil"
)

func LoadToppingPriceFixtures(t *testing.T, db *gorm.DB, restaurantID uuid.UUID) []readmodel.ToppingPrice {
	prices := []readmodel.ToppingPrice{
		{
			RestaurantID: restaurantID,
			ToppingID:    testutil.MustNewID(),
			Name:         "Extra Cheese",
			ExtraPrice:   decimal.NewFromFloat(1.50),
			UpdatedAt:    time.Now().UTC(),
		},
		{
			RestaurantID: restaurantID,
			ToppingID:    testutil.MustNewID(),
			Name:         "Mushrooms",
			ExtraPrice:   decimal.NewFromFloat(1.00),
			UpdatedAt:    time.Now().UTC(),
		},
	}

	require.NoError(t, db.Create(&prices).Error)

	return prices
}
