package testutil

import (
	"context"

	"github.com/google/uuid"

	"order-service/internal/domain/readmodel"
)

type MockCustomerRepository struct {
	FindByIDResult *readmodel.Customer
	FindByIDErr    error
	Upserted       []readmodel.Customer
	UpsertErr      error
}

var _ readmodel.CustomerRepository = (*MockCustomerRepository)(nil)

func (m *MockCustomerRepository) FindByID(_ context.Context, _ uuid.UUID) (*readmodel.Customer, error) {
	return m.FindByIDResult, m.FindByIDErr
}

func (m *MockCustomerRepository) Upsert(_ context.Context, customer readmodel.Customer) error {
	m.Upserted = append(m.Upserted, customer)
	return m.UpsertErr
}
