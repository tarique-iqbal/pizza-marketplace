package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/cart"
	apperr "order-service/internal/shared/errors"
)

type CartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) cart.CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) FindByCustomer(ctx context.Context, customerID uuid.UUID) (*cart.Cart, error) {
	var c cart.Cart

	err := r.db.WithContext(ctx).Preload("Items").First(&c, "customer_id = ?", customerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *CartRepository) Create(ctx context.Context, c *cart.Cart) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CartRepository) AddOrMergeItem(ctx context.Context, cartID uuid.UUID, item cart.CartItem) error {
	item.CartID = cartID

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "cart_id"}, {Name: "pizza_id"}, {Name: "size_id"}, {Name: "toppings"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity": gorm.Expr("cart_items.quantity + excluded.quantity"),
		}),
	}).Create(&item).Error
}

func (r *CartRepository) UpdateItemQuantity(ctx context.Context, cartID, itemID uuid.UUID, quantity int16) error {
	result := r.db.WithContext(ctx).Model(&cart.CartItem{}).
		Where("id = ? AND cart_id = ?", itemID, cartID).
		Update("quantity", quantity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

func (r *CartRepository) RemoveItem(ctx context.Context, cartID, itemID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ? AND cart_id = ?", itemID, cartID).Delete(&cart.CartItem{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperr.ErrNotFound
	}

	return nil
}

func (r *CartRepository) Clear(ctx context.Context, cartID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", cartID).Delete(&cart.Cart{}).Error
}
