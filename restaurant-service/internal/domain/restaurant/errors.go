package restaurant

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrPendingPayoutExists = errors.New("pending payout already exists")
)
