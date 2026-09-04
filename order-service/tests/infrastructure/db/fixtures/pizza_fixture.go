package fixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"order-service/internal/domain/readmodel"
	"order-service/tests/testutil"
)

func LoadPizzaFixtures(t *testing.T, db *gorm.DB, restaurantID uuid.UUID) []readmodel.Pizza {
	pizzas := []readmodel.Pizza{
		{
			ID:           testutil.MustNewID(),
			RestaurantID: restaurantID,
			Name:         "Margherita",
			Status:       readmodel.PizzaAvailable,
			UpdatedAt:    time.Now().UTC(),
		},
		{
			ID:           testutil.MustNewID(),
			RestaurantID: restaurantID,
			Name:         "Quattro Stagioni",
			Status:       readmodel.PizzaUnavailable,
			UpdatedAt:    time.Now().UTC(),
		},
	}

	require.NoError(t, db.Create(&pizzas).Error)

	return pizzas
}
