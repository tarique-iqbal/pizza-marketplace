package auth

import (
	"context"
	"errors"
)

var (
	ErrCodeInvalid     = errors.New("invalid verification code")
	ErrCodeExpired     = errors.New("verification code expired")
	ErrCodeUsed        = errors.New("verification code already used")
	ErrCodeNotIssued   = errors.New("verification code not issued")
	ErrTooManyAttempts = errors.New("too many failed verification attempts")
)

type EmailVerifier interface {
	Verify(ctx context.Context, email string, code string) error
}
