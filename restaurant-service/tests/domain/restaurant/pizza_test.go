package restaurant

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"restaurant-service/internal/domain/restaurant"
)

func TestPizza_SetToppingIDs_RoundTrip(t *testing.T) {
	pizza := restaurant.NewPizza(uuid.New(), uuid.New())

	ids := []uuid.UUID{uuid.New(), uuid.New()}
	require.NoError(t, pizza.SetToppingIDs(ids))

	stored, err := pizza.ToppingIDs()
	require.NoError(t, err)
	assert.Equal(t, ids, stored)
}

func TestPizza_SetToppingIDs_RejectsDuplicates(t *testing.T) {
	pizza := restaurant.NewPizza(uuid.New(), uuid.New())

	dup := uuid.New()

	err := pizza.SetToppingIDs([]uuid.UUID{dup, dup})
	require.Error(t, err)

	assert.ErrorIs(t, err, restaurant.ErrDuplicateTopping)
}

func TestPizza_ToppingIDs_EmptyByDefault(t *testing.T) {
	pizza := restaurant.NewPizza(uuid.New(), uuid.New())

	ids, err := pizza.ToppingIDs()
	require.NoError(t, err)
	assert.Empty(t, ids)
}
