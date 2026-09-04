package testutil

import (
	"context"

	"github.com/google/uuid"

	"order-service/internal/domain/readmodel"
)

type MockToppingPriceRepository struct {
	ListByRestaurantResult []readmodel.ToppingPrice
	ListByRestaurantErr    error
	Upserted               []readmodel.ToppingPrice
	UpsertErr              error
}

var _ readmodel.ToppingPriceRepository = (*MockToppingPriceRepository)(nil)

func (m *MockToppingPriceRepository) ListByRestaurant(
	_ context.Context,
	_ uuid.UUID,
) ([]readmodel.ToppingPrice, error) {
	return m.ListByRestaurantResult, m.ListByRestaurantErr
}

func (m *MockToppingPriceRepository) Upsert(_ context.Context, price readmodel.ToppingPrice) error {
	m.Upserted = append(m.Upserted, price)
	return m.UpsertErr
}
