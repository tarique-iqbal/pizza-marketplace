package payout

import "errors"

var (
	ErrUnverifiedPayoutExists = errors.New("unverified payout already exists")
	ErrNoUnverifiedPayout     = errors.New("no unverified payout to update")
)
