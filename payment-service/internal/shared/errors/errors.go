package errors

import "errors"

var (
	ErrUnauthorized = errors.New("unauthorized access")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("conflict")
	ErrInvalid      = errors.New("invalid input")
)
