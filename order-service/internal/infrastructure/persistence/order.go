package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/order"
	apperr "order-service/internal/shared/errors"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) order.OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *OrderRepository) Update(ctx context.Context, o *order.Order) error {
	return r.db.WithContext(ctx).Omit(clause.Associations).Save(o).Error
}

func (r *OrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	var o order.Order

	err := r.db.WithContext(ctx).Preload("Items").First(&o, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *OrderRepository) FindByIDAndCustomer(
	ctx context.Context,
	id, customerID uuid.UUID,
) (*order.Order, error) {
	var o order.Order

	err := r.db.WithContext(ctx).Preload("Items").
		First(&o, "id = ? AND customer_id = ?", id, customerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *OrderRepository) FindByIDAndRestaurantOwner(
	ctx context.Context,
	id, ownerID uuid.UUID,
) (*order.Order, error) {
	var o order.Order

	err := r.db.WithContext(ctx).Preload("Items").
		Joins("JOIN restaurants ON restaurants.id = orders.restaurant_id").
		Where("orders.id = ? AND restaurants.owner_id = ?", id, ownerID).
		First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]order.Order, error) {
	var orders []order.Order

	err := r.db.WithContext(ctx).Preload("Items").
		Where("customer_id = ?", customerID).
		Order("placed_at DESC").
		Find(&orders).Error

	return orders, err
}

func (r *OrderRepository) ListByRestaurantOwner(ctx context.Context, ownerID uuid.UUID) ([]order.Order, error) {
	var orders []order.Order

	err := r.db.WithContext(ctx).Preload("Items").
		Joins("JOIN restaurants ON restaurants.id = orders.restaurant_id").
		Where("restaurants.owner_id = ?", ownerID).
		Order("orders.placed_at DESC").
		Find(&orders).Error

	return orders, err
}
