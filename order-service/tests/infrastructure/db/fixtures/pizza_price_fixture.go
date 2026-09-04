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

func LoadPizzaPriceFixtures(t *testing.T, db *gorm.DB, pizzaID uuid.UUID) []readmodel.PizzaPrice {
	prices := []readmodel.PizzaPrice{
		{
			PizzaID:    pizzaID,
			SizeID:     testutil.MustNewID(),
			DiameterCm: 26,
			Price:      decimal.NewFromFloat(7.50),
			IsActive:   true,
			UpdatedAt:  time.Now().UTC(),
		},
		{
			PizzaID:    pizzaID,
			SizeID:     testutil.MustNewID(),
			DiameterCm: 32,
			Price:      decimal.NewFromFloat(10.50),
			IsActive:   true,
			UpdatedAt:  time.Now().UTC(),
		},
	}

	require.NoError(t, db.Create(&prices).Error)

	return prices
}
