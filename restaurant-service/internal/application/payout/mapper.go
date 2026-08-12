package payout

import (
	"restaurant-service/internal/domain/payout"
)

func ToPayoutResponse(pd *payout.PayoutDetails) PayoutResponse {
	if pd == nil {
		return PayoutResponse{}
	}

	return PayoutResponse{
		AccountHolder: pd.AccountHolder,
		IBAN:          pd.IBAN,
		BIC:           pd.BIC,
		BankName:      pd.BankName,
		Status:        pd.Status,
	}
}
