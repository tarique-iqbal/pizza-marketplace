package restaurant

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrChecklistIncomplete = errors.New("restaurant onboarding checklist is not complete")
)
