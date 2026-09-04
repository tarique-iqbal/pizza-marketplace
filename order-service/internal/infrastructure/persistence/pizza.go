package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"order-service/internal/domain/readmodel"
	apperr "order-service/internal/shared/errors"
)

type PizzaRepository struct {
	db *gorm.DB
}

func NewPizzaRepository(db *gorm.DB) readmodel.PizzaRepository {
	return &PizzaRepository{db: db}
}

func (r *PizzaRepository) FindByID(ctx context.Context, id uuid.UUID) (*readmodel.Pizza, error) {
	var pizza readmodel.Pizza

	err := r.db.WithContext(ctx).First(&pizza, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &pizza, nil
}

// Upsert is guarded by updatedAt — a no-op if the stored row is already newer.
func (r *PizzaRepository) Upsert(ctx context.Context, pizza readmodel.Pizza) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"restaurant_id", "name", "status", "updated_at"}),
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "pizzas.updated_at < excluded.updated_at"},
		}},
	}).Create(&pizza).Error
}

func (r *PizzaRepository) Delete(ctx context.Context, id uuid.UUID, updatedAt time.Time) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND updated_at < ?", id, updatedAt).
		Delete(&readmodel.Pizza{}).Error
}
