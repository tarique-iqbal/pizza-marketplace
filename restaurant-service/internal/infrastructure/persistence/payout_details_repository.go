package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"restaurant-service/internal/domain/payout"
)

const pgUniqueViolation = "23505"

type PayoutDetailsRepository struct {
	db *gorm.DB
}

func NewPayoutDetailsRepository(db *gorm.DB) payout.PayoutDetailsRepository {
	return &PayoutDetailsRepository{db: db}
}

func (r *PayoutDetailsRepository) WithTx(tx *gorm.DB) payout.PayoutDetailsRepository {
	return &PayoutDetailsRepository{db: tx}
}

func (repo *PayoutDetailsRepository) Create(
	ctx context.Context,
	pd *payout.PayoutDetails,
) error {
	if err := repo.db.WithContext(ctx).Create(pd).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return payout.ErrPendingPayoutExists
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
		Model(&payout.PayoutDetails{}).
		Where("restaurant_id = ? AND status = ?", restaurantID, payout.PayoutPending).
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
		return payout.ErrNoPendingPayout
	}

	return nil
}

func (repo *PayoutDetailsRepository) PromoteToActive(
	ctx context.Context,
	restaurantID uuid.UUID,
) error {
	result := repo.db.WithContext(ctx).
		Model(&payout.PayoutDetails{}).
		Where("restaurant_id = ? AND status = ?", restaurantID, payout.PayoutPending).
		Update("status", payout.PayoutActive)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return payout.ErrNoPendingPayout
	}

	return nil
}

func (repo *PayoutDetailsRepository) FindActiveByRestaurant(
	ctx context.Context,
	restaurantID uuid.UUID,
) (*payout.PayoutDetails, error) {
	var pd payout.PayoutDetails

	err := repo.db.WithContext(ctx).
		Where("restaurant_id = ? AND status = ?", restaurantID, payout.PayoutActive).
		Take(&pd).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &pd, err
}
