package persistence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/restaurant"
)

const pgUniqueViolation = "23505"

type PayoutDetailsRepository struct {
	db *gorm.DB
}

func NewPayoutDetailsRepository(db *gorm.DB) restaurant.PayoutDetailsRepository {
	return &PayoutDetailsRepository{db: db}
}

func (r *PayoutDetailsRepository) WithTx(tx *gorm.DB) restaurant.PayoutDetailsRepository {
	return &PayoutDetailsRepository{db: tx}
}

func (repo *PayoutDetailsRepository) Create(
	ctx context.Context,
	pd *restaurant.PayoutDetails,
) error {
	if err := repo.db.WithContext(ctx).Create(pd).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return restaurant.ErrPendingPayoutExists
		}
		return err
	}

	return nil
}
