package restaurant

import (
	"time"

	"github.com/google/uuid"
)

type PayoutStatus string

const (
	PayoutPending    PayoutStatus = "pending"
	PayoutActive     PayoutStatus = "active"
	PayoutSuperseded PayoutStatus = "superseded"
)

type PayoutDetails struct {
	ID            uuid.UUID    `gorm:"type:uuid;primaryKey"`
	RestaurantID  uuid.UUID    `gorm:"column:restaurant_id;type:uuid;not null;index"`
	AccountHolder string       `gorm:"column:account_holder;size:100;not null"`
	IBAN          string       `gorm:"column:iban;size:34;not null"`
	BIC           string       `gorm:"column:bic;size:11;not null"`
	BankName      string       `gorm:"column:bank_name;size:100;not null"`
	Status        PayoutStatus `gorm:"type:payout_details_status_enum;not null;default:'pending'"`
	CreatedAt     time.Time    `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt     *time.Time   `gorm:"type:timestamptz"`
}

func (PayoutDetails) TableName() string {
	return "payout_details"
}

func NewPayoutDetails(
	restaurantID uuid.UUID,
	accountHolder string,
	iban string,
	bic string,
	bankName string,
) (*PayoutDetails, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &PayoutDetails{
		ID:            id,
		RestaurantID:  restaurantID,
		AccountHolder: accountHolder,
		IBAN:          iban,
		BIC:           bic,
		BankName:      bankName,
		Status:        PayoutPending,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
