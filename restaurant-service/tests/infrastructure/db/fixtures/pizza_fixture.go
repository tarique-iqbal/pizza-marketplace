package fixtures

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
	"restaurant-service/tests/testutil"
)

func LoadPizzaFixtures(t *testing.T, db *gorm.DB) error {
	var owner restaurant.Restaurant

	err := db.Where("slug = ?", "anatolische-kueche").Take(&owner).Error
	require.NoError(t, err)

	pizzas := []restaurant.Pizza{
		{
			ID:           testutil.MustNewID(),
			RestaurantID: owner.ID,
			Name:         "Margherita",
			IsVegetarian: true,
			Status:       restaurant.PizzaAvailable,
			SortOrder:    1,
			CreatedAt:    time.Now().UTC(),
		},
		{
			ID:           testutil.MustNewID(),
			RestaurantID: owner.ID,
			Name:         "Salami",
			IsVegetarian: false,
			Status:       restaurant.PizzaAvailable,
			SortOrder:    2,
			CreatedAt:    time.Now().UTC(),
		},
	}

	for _, p := range pizzas {
		err := db.Create(&p).Error
		require.NoError(t, err)
	}

	return nil
}
