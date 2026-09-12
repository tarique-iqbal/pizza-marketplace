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

	ListByCustomerResult   []order.Order
	ListByCustomerErr      error
	ListByCustomerCalls    []ListOrdersCall
	ListByRestaurantResult []order.Order
	ListByRestaurantErr    error
	ListByRestaurantCalls  []ListByRestaurantCall
}

type ListOrdersCall struct {
	ID    uuid.UUID
	After *order.PageCursor
	Limit int
}

type ListByRestaurantCall struct {
	RestaurantID uuid.UUID
	OwnerID      uuid.UUID
	After        *order.PageCursor
	Limit        int
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

func (m *MockOrderRepository) ListByCustomer(
	_ context.Context,
	customerID uuid.UUID,
	after *order.PageCursor,
	limit int,
) ([]order.Order, error) {
	m.ListByCustomerCalls = append(m.ListByCustomerCalls, ListOrdersCall{ID: customerID, After: after, Limit: limit})
	return m.ListByCustomerResult, m.ListByCustomerErr
}

func (m *MockOrderRepository) ListByRestaurant(
	_ context.Context,
	restaurantID, ownerID uuid.UUID,
	after *order.PageCursor,
	limit int,
) ([]order.Order, error) {
	m.ListByRestaurantCalls = append(
		m.ListByRestaurantCalls,
		ListByRestaurantCall{RestaurantID: restaurantID, OwnerID: ownerID, After: after, Limit: limit},
	)
	return m.ListByRestaurantResult, m.ListByRestaurantErr
}
