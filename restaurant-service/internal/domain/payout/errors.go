package payout

import "errors"

var (
	ErrPendingPayoutExists = errors.New("pending payout already exists")
	ErrNoPendingPayout     = errors.New("no pending payout to update")
)
