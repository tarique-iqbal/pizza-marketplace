package testutil

import (
	"context"

	"github.com/google/uuid"

	"customer-service/internal/domain/customer"
)

type MockCustomerRepository struct {
	FindByIDResult *customer.Customer
	FindByIDErr    error
	Upserted       []customer.Customer
	UpsertErr      error
	UpdateErr      error
}

var _ customer.CustomerRepository = (*MockCustomerRepository)(nil)

func (m *MockCustomerRepository) FindByID(_ context.Context, _ uuid.UUID) (*customer.Customer, error) {
	return m.FindByIDResult, m.FindByIDErr
}

func (m *MockCustomerRepository) Upsert(_ context.Context, c customer.Customer) error {
	m.Upserted = append(m.Upserted, c)
	return m.UpsertErr
}

func (m *MockCustomerRepository) Update(_ context.Context, _ *customer.Customer) error {
	return m.UpdateErr
}
