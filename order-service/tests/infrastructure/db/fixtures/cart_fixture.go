package fixtures

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"order-service/internal/domain/cart"
	"order-service/tests/testutil"
)

func LoadCartFixtures(t *testing.T, db *gorm.DB) []cart.Cart {
	carts := []cart.Cart{
		{
			ID:           testutil.MustNewID(),
			CustomerID:   testutil.MustNewID(),
			RestaurantID: testutil.MustNewID(),
			Items: []cart.CartItem{
				{
					ID:              testutil.MustNewID(),
					PizzaID:         testutil.MustNewID(),
					SizeID:          testutil.MustNewID(),
					Quantity:        2,
					ExtraToppingIDs: []uuid.UUID{},
				},
			},
		},
		{
			ID:           testutil.MustNewID(),
			CustomerID:   testutil.MustNewID(),
			RestaurantID: testutil.MustNewID(),
		},
	}

	require.NoError(t, db.Create(&carts).Error)

	return carts
}
