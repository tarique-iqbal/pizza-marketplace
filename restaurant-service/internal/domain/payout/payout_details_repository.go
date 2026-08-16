package payout

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PayoutDetailsRepository interface {
	WithTx(tx *gorm.DB) PayoutDetailsRepository
	Create(ctx context.Context, pd *PayoutDetails) error
	UpdatePending(
		ctx context.Context,
		restaurantID uuid.UUID,
		accountHolder string,
		iban string,
		bic string,
		bankName string,
	) error
	FindActiveByRestaurant(ctx context.Context, restaurantID uuid.UUID) (*PayoutDetails, error)
	PromoteToActive(ctx context.Context, restaurantID uuid.UUID) error
}
