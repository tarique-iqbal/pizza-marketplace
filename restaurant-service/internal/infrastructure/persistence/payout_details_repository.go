package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
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

func (repo *PayoutDetailsRepository) UpdatePending(
	ctx context.Context,
	restaurantID uuid.UUID,
	accountHolder string,
	iban string,
	bic string,
	bankName string,
) error {
	result := repo.db.WithContext(ctx).
		Model(&restaurant.PayoutDetails{}).
		Where("restaurant_id = ? AND status = ?", restaurantID, restaurant.PayoutPending).
		Updates(map[string]any{
			"account_holder": accountHolder,
			"iban":           iban,
			"bic":            bic,
			"bank_name":      bankName,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return restaurant.ErrNoPendingPayout
	}

	return nil
}
