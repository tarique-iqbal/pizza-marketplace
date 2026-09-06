package cart_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"order-service/internal/domain/cart"
)

func TestNewCart(t *testing.T) {
	id := uuid.New()
	customerID := uuid.New()
	restaurantID := uuid.New()

	c := cart.NewCart(id, customerID, restaurantID)

	assert.Equal(t, id, c.ID)
	assert.Equal(t, customerID, c.CustomerID)
	assert.Equal(t, restaurantID, c.RestaurantID)
	assert.Empty(t, c.Items)
}

func TestCart_EnsureRestaurant_SameRestaurant(t *testing.T) {
	restaurantID := uuid.New()
	c := cart.Cart{RestaurantID: restaurantID}

	err := c.EnsureRestaurant(restaurantID)
	require.NoError(t, err)
}

func TestCart_EnsureRestaurant_DifferentRestaurant(t *testing.T) {
	c := cart.Cart{RestaurantID: uuid.New()}

	err := c.EnsureRestaurant(uuid.New())

	require.ErrorIs(t, err, cart.ErrCartRestaurantMismatch)
}
