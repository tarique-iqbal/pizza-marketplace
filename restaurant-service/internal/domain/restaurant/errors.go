package restaurant

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrPendingPayoutExists = errors.New("pending payout already exists")
	ErrNoPendingPayout     = errors.New("no pending payout to update")
)
