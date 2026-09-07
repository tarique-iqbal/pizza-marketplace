package fixtures

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"order-service/internal/domain/order"
	"order-service/internal/domain/readmodel"
	"order-service/tests/testutil"
)

func LoadOrderFixtures(t *testing.T, db *gorm.DB, restaurants []readmodel.Restaurant) []order.Order {
	orders := []order.Order{
		{
			ID:           testutil.MustNewID(),
			CustomerID:   testutil.MustNewID(),
			RestaurantID: restaurants[0].ID,
			Status:       order.StatusPending,
			Fulfillment:  order.FulfillmentPickup,
			ContactEmail: "customer.one@example.com",
			Items: []order.OrderItem{
				{
					ID:           testutil.MustNewID(),
					PizzaID:      testutil.MustNewID(),
					SizeID:       testutil.MustNewID(),
					PizzaName:    "Margherita",
					SizeDiameter: 26,
					Quantity:     2,
					UnitPrice:    decimal.NewFromFloat(7.50),
					TotalPrice:   decimal.NewFromFloat(15.00),
				},
			},
			Subtotal:    decimal.NewFromFloat(15.00),
			DeliveryFee: decimal.Zero,
			Total:       decimal.NewFromFloat(15.00),
			Currency:    "EUR",
			PlacedAt:    time.Now().UTC(),
		},
		{
			ID:           testutil.MustNewID(),
			CustomerID:   testutil.MustNewID(),
			RestaurantID: restaurants[1].ID,
			Status:       order.StatusPending,
			Fulfillment:  order.FulfillmentPickup,
			ContactEmail: "customer.two@example.com",
			Items: []order.OrderItem{
				{
					ID:           testutil.MustNewID(),
					PizzaID:      testutil.MustNewID(),
					SizeID:       testutil.MustNewID(),
					PizzaName:    "Lahmacun",
					SizeDiameter: 24,
					Quantity:     1,
					UnitPrice:    decimal.NewFromFloat(6.00),
					TotalPrice:   decimal.NewFromFloat(6.00),
				},
			},
			Subtotal:    decimal.NewFromFloat(6.00),
			DeliveryFee: decimal.Zero,
			Total:       decimal.NewFromFloat(6.00),
			Currency:    "EUR",
			PlacedAt:    time.Now().UTC(),
		},
	}

	require.NoError(t, db.Create(&orders).Error)

	return orders
}
