package payment

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid payment status transition")
)
