package testutil

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"order-service/internal/domain/order"
)

type MockOrderRepository struct {
	Created   []order.Order
	CreateErr error

	Updated   []order.Order
	UpdateErr error

	FindByIDResult                   *order.Order
	FindByIDErr                      error
	FindByIDAndCustomerResult        *order.Order
	FindByIDAndCustomerErr           error
	FindByIDAndRestaurantOwnerResult *order.Order
	FindByIDAndRestaurantOwnerErr    error

	ListByCustomerResult        []order.Order
	ListByCustomerErr           error
	ListByRestaurantOwnerResult []order.Order
	ListByRestaurantOwnerErr    error
}

var _ order.OrderRepository = (*MockOrderRepository)(nil)

func (m *MockOrderRepository) WithTx(_ *gorm.DB) order.OrderRepository {
	return m
}

func (m *MockOrderRepository) Create(_ context.Context, o *order.Order) error {
	m.Created = append(m.Created, *o)
	return m.CreateErr
}

func (m *MockOrderRepository) Update(_ context.Context, o *order.Order) error {
	m.Updated = append(m.Updated, *o)
	return m.UpdateErr
}

func (m *MockOrderRepository) FindByID(_ context.Context, _ uuid.UUID) (*order.Order, error) {
	return m.FindByIDResult, m.FindByIDErr
}

func (m *MockOrderRepository) FindByIDAndCustomer(
	_ context.Context,
	_, _ uuid.UUID,
) (*order.Order, error) {
	return m.FindByIDAndCustomerResult, m.FindByIDAndCustomerErr
}

func (m *MockOrderRepository) FindByIDAndRestaurantOwner(
	_ context.Context,
	_, _ uuid.UUID,
) (*order.Order, error) {
	return m.FindByIDAndRestaurantOwnerResult, m.FindByIDAndRestaurantOwnerErr
}

func (m *MockOrderRepository) ListByCustomer(_ context.Context, _ uuid.UUID) ([]order.Order, error) {
	return m.ListByCustomerResult, m.ListByCustomerErr
}

func (m *MockOrderRepository) ListByRestaurantOwner(_ context.Context, _ uuid.UUID) ([]order.Order, error) {
	return m.ListByRestaurantOwnerResult, m.ListByRestaurantOwnerErr
}
