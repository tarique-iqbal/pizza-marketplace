package auth

import (
	"context"
	"errors"
	"identity-service/internal/domain/auth"
	"time"
)

const maxVerificationAttempts = 3

type emailVerifier struct {
	repo auth.EmailVerificationRepository
}

func NewEmailVerifier(
	repo auth.EmailVerificationRepository,
) auth.EmailVerifier {
	return &emailVerifier{repo: repo}
}

func (s *emailVerifier) Verify(ctx context.Context, email string, code string) error {
	emailVerification, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return auth.ErrCodeInvalid
	}

	if emailVerification == nil {
		return auth.ErrCodeNotIssued
	}

	if emailVerification.IsUsed {
		return auth.ErrCodeUsed
	}

	if emailVerification.AttemptCount >= maxVerificationAttempts {
		return auth.ErrTooManyAttempts
	}

	if emailVerification.Code != code {
		if err := s.repo.IncrementAttempts(ctx, emailVerification.ID); err != nil {
			return err
		}
		return auth.ErrCodeInvalid
	}

	if time.Now().UTC().After(emailVerification.ExpiresAt) {
		return auth.ErrCodeExpired
	}

	emailVerification.Code = "..."
	emailVerification.IsUsed = true
	if err := s.repo.Updates(ctx, emailVerification); err != nil {
		return errors.New("failed to mark code as used")
	}

	return nil
}
