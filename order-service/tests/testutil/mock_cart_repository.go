package testutil

import (
	"context"

	"github.com/google/uuid"

	"order-service/internal/domain/cart"
)

type AddOrMergeItemCall struct {
	CartID uuid.UUID
	Item   cart.CartItem
}

type UpdateItemQuantityCall struct {
	CartID   uuid.UUID
	ItemID   uuid.UUID
	Quantity int16
}

type RemoveItemCall struct {
	CartID uuid.UUID
	ItemID uuid.UUID
}

type MockCartRepository struct {
	FindByCustomerResult *cart.Cart
	FindByCustomerErr    error

	Created   []cart.Cart
	CreateErr error

	AddOrMergeItemCalls []AddOrMergeItemCall
	AddOrMergeItemErr   error

	UpdateItemQuantityCalls []UpdateItemQuantityCall
	UpdateItemQuantityErr   error

	RemoveItemCalls []RemoveItemCall
	RemoveItemErr   error

	ClearedCartIDs []uuid.UUID
	ClearErr       error
}

var _ cart.CartRepository = (*MockCartRepository)(nil)

func (m *MockCartRepository) FindByCustomer(_ context.Context, _ uuid.UUID) (*cart.Cart, error) {
	return m.FindByCustomerResult, m.FindByCustomerErr
}

func (m *MockCartRepository) Create(_ context.Context, c *cart.Cart) error {
	m.Created = append(m.Created, *c)
	return m.CreateErr
}

func (m *MockCartRepository) AddOrMergeItem(_ context.Context, cartID uuid.UUID, item cart.CartItem) error {
	m.AddOrMergeItemCalls = append(m.AddOrMergeItemCalls, AddOrMergeItemCall{CartID: cartID, Item: item})
	return m.AddOrMergeItemErr
}

func (m *MockCartRepository) UpdateItemQuantity(
	_ context.Context,
	cartID, itemID uuid.UUID,
	quantity int16,
) error {
	m.UpdateItemQuantityCalls = append(
		m.UpdateItemQuantityCalls,
		UpdateItemQuantityCall{CartID: cartID, ItemID: itemID, Quantity: quantity},
	)
	return m.UpdateItemQuantityErr
}

func (m *MockCartRepository) RemoveItem(_ context.Context, cartID, itemID uuid.UUID) error {
	m.RemoveItemCalls = append(m.RemoveItemCalls, RemoveItemCall{CartID: cartID, ItemID: itemID})
	return m.RemoveItemErr
}

func (m *MockCartRepository) Clear(_ context.Context, cartID uuid.UUID) error {
	m.ClearedCartIDs = append(m.ClearedCartIDs, cartID)
	return m.ClearErr
}
