package restaurant

import "errors"

var (
	ErrEmailAlreadyExists           = errors.New("email already exists")
	ErrChecklistIncomplete          = errors.New("restaurant onboarding checklist is not complete")
	ErrNotPendingReview             = errors.New("restaurant is not pending review")
	ErrNotReadyToLaunch             = errors.New("restaurant is not ready to launch")
	ErrLaunchReadinessNotApplicable = errors.New(
		"launch readiness is not accessible once a restaurant has launched",
	)
	ErrNotEnoughPizzas = errors.New("restaurant does not have enough priced pizzas to launch")
)
