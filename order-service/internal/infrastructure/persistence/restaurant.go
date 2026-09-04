package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
)

type RestaurantRepository struct {
	db *gorm.DB
}

func NewRestaurantRepository(db *gorm.DB) readmodel.RestaurantRepository {
	return &RestaurantRepository{db: db}
}

func (r *RestaurantRepository) FindByID(ctx context.Context, id uuid.UUID) (*readmodel.Restaurant, error) {
	var restaurant readmodel.Restaurant

	err := r.db.WithContext(ctx).First(&restaurant, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &restaurant, nil
}

// Upsert is guarded by updatedAt — a no-op if the stored row is already newer.
func (r *RestaurantRepository) Upsert(ctx context.Context, restaurant readmodel.Restaurant) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"owner_id", "name", "owner_email", "lat", "lon", "delivery_km",
			"delivery_fee", "minimum_order", "pickup", "delivery_type", "currency", "updated_at",
		}),
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "restaurants.updated_at < excluded.updated_at"},
		}},
	}).Create(&restaurant).Error
}
