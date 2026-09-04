package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"

	"order-service/internal/domain/readmodel"
)

type DeletedPizza struct {
	ID        uuid.UUID
	UpdatedAt time.Time
}

type MockPizzaRepository struct {
	FindByIDResult *readmodel.Pizza
	FindByIDErr    error
	Upserted       []readmodel.Pizza
	UpsertErr      error
	Deleted        []DeletedPizza
	DeleteErr      error
}

var _ readmodel.PizzaRepository = (*MockPizzaRepository)(nil)

func (m *MockPizzaRepository) FindByID(_ context.Context, _ uuid.UUID) (*readmodel.Pizza, error) {
	return m.FindByIDResult, m.FindByIDErr
}

func (m *MockPizzaRepository) Upsert(_ context.Context, pizza readmodel.Pizza) error {
	m.Upserted = append(m.Upserted, pizza)
	return m.UpsertErr
}

func (m *MockPizzaRepository) Delete(_ context.Context, id uuid.UUID, updatedAt time.Time) error {
	m.Deleted = append(m.Deleted, DeletedPizza{ID: id, UpdatedAt: updatedAt})
	return m.DeleteErr
}
