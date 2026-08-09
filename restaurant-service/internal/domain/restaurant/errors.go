package restaurant

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrPendingPayoutExists = errors.New("pending payout already exists")
	ErrNoPendingPayout     = errors.New("no pending payout to update")
	ErrChecklistIncomplete = errors.New("restaurant onboarding checklist is not complete")
	ErrDuplicateTopping    = errors.New("duplicate topping in selection")
)
