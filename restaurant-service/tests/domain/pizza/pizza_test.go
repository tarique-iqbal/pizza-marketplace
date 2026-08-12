package pizza_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"restaurant-service/internal/domain/pizza"
)

func TestPizza_SetToppingIDs_RoundTrip(t *testing.T) {
	p := pizza.NewPizza(uuid.New(), uuid.New())

	ids := []uuid.UUID{uuid.New(), uuid.New()}
	require.NoError(t, p.SetToppingIDs(ids))

	stored, err := p.ToppingIDs()
	require.NoError(t, err)
	assert.Equal(t, ids, stored)
}

func TestPizza_SetToppingIDs_RejectsDuplicates(t *testing.T) {
	p := pizza.NewPizza(uuid.New(), uuid.New())

	dup := uuid.New()

	err := p.SetToppingIDs([]uuid.UUID{dup, dup})
	require.Error(t, err)

	assert.ErrorIs(t, err, pizza.ErrDuplicateTopping)
}

func TestPizza_ToppingIDs_EmptyByDefault(t *testing.T) {
	p := pizza.NewPizza(uuid.New(), uuid.New())

	ids, err := p.ToppingIDs()
	require.NoError(t, err)
	assert.Empty(t, ids)
}
