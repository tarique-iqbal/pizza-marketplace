package payout

import (
	"restaurant-service/internal/domain/payout"
)

type PayoutStatus = payout.PayoutStatus

type CreatePayoutRequest struct {
	AccountHolder string `json:"accountHolder" binding:"required,max=100"`
	IBAN          string `json:"iban" binding:"required,iban"`
	BIC           string `json:"bic" binding:"required,bic"`
	BankName      string `json:"bankName" binding:"required,max=100"`
}

type UpdatePayoutRequest = CreatePayoutRequest

type PayoutResponse struct {
	AccountHolder string       `json:"accountHolder"`
	IBAN          string       `json:"iban"`
	BIC           string       `json:"bic"`
	BankName      string       `json:"bankName"`
	Status        PayoutStatus `json:"status,omitempty"`
}
