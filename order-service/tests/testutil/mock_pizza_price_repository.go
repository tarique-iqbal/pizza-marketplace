package testutil

import (
	"context"

	"github.com/google/uuid"

	"order-service/internal/domain/readmodel"
)

type MockPizzaPriceRepository struct {
	ListByPizzaResult []readmodel.PizzaPrice
	ListByPizzaErr    error
	Upserted          []readmodel.PizzaPrice
	UpsertErr         error
}

var _ readmodel.PizzaPriceRepository = (*MockPizzaPriceRepository)(nil)

func (m *MockPizzaPriceRepository) ListByPizza(_ context.Context, _ uuid.UUID) ([]readmodel.PizzaPrice, error) {
	return m.ListByPizzaResult, m.ListByPizzaErr
}

func (m *MockPizzaPriceRepository) Upsert(_ context.Context, price readmodel.PizzaPrice) error {
	m.Upserted = append(m.Upserted, price)
	return m.UpsertErr
}
