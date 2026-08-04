package restaurant

import (
	"context"

	"gorm.io/gorm"
)

type PayoutDetailsRepository interface {
	WithTx(tx *gorm.DB) PayoutDetailsRepository
	Create(ctx context.Context, pd *PayoutDetails) error
}
