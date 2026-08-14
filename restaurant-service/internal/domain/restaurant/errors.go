package restaurant

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrChecklistIncomplete = errors.New("restaurant onboarding checklist is not complete")
	ErrNotPendingReview    = errors.New("restaurant is not pending review")
	ErrNotReadyToLaunch    = errors.New("restaurant is not ready to launch")
)
