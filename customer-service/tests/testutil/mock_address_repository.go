package testutil

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"customer-service/internal/domain/customer"
)

type MockAddressRepository struct {
	Created              []customer.Address
	CreateErr            error
	DeleteErr            error
	FindByIDResult       *customer.Address
	FindByIDErr          error
	ListByCustomerResult []customer.Address
	ListByCustomerErr    error
	UnsetDefaultErr      error
	SetDefaultErr        error
	ExistsResult         bool
	ExistsErr            error
}

var _ customer.AddressRepository = (*MockAddressRepository)(nil)

func (m *MockAddressRepository) WithTx(_ *gorm.DB) customer.AddressRepository {
	return m
}

func (m *MockAddressRepository) Create(_ context.Context, a *customer.Address) error {
	m.Created = append(m.Created, *a)
	return m.CreateErr
}

func (m *MockAddressRepository) Delete(_ context.Context, _, _ uuid.UUID) error {
	return m.DeleteErr
}

func (m *MockAddressRepository) FindByID(_ context.Context, _, _ uuid.UUID) (*customer.Address, error) {
	return m.FindByIDResult, m.FindByIDErr
}

func (m *MockAddressRepository) ListByCustomer(_ context.Context, _ uuid.UUID) ([]customer.Address, error) {
	return m.ListByCustomerResult, m.ListByCustomerErr
}

func (m *MockAddressRepository) UnsetDefault(_ context.Context, _ uuid.UUID) error {
	return m.UnsetDefaultErr
}

func (m *MockAddressRepository) SetDefault(_ context.Context, _ uuid.UUID) error {
	return m.SetDefaultErr
}

func (m *MockAddressRepository) ExistsForCustomer(
	_ context.Context,
	_ uuid.UUID,
	_, _, _, _ string,
) (bool, error) {
	return m.ExistsResult, m.ExistsErr
}
