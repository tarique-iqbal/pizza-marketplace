package auth

import (
	"context"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/user"
	"identity-service/internal/shared/event"
	"strings"
	"time"
)

const accessTokenExpiry = 15

type RequestEmailOTP struct {
	repo      auth.EmailVerificationRepository
	userRepo  user.UserRepository
	otp       auth.OTPGenerator
	publisher event.EventPublisher
}

func NewRequestEmailOTP(
	repo auth.EmailVerificationRepository,
	userRepo user.UserRepository,
	otp auth.OTPGenerator,
	publisher event.EventPublisher,
) *RequestEmailOTP {
	return &RequestEmailOTP{repo: repo, userRepo: userRepo, otp: otp, publisher: publisher}
}

func (uc *RequestEmailOTP) Execute(
	ctx context.Context,
	input EmailVerificationRequest,
) error {
	email := strings.ToLower(input.Email)

	exists, err := uc.userRepo.EmailExists(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return user.ErrEmailAlreadyExists
	}

	code, err := uc.otp.Generate(true)
	if err != nil {
		return err
	}

	expiry := time.Duration(accessTokenExpiry) * time.Minute
	verification := &auth.EmailVerification{
		Email:     email,
		Code:      code,
		IsUsed:    false,
		ExpiresAt: time.Now().UTC().Add(expiry),
	}

	existing, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	if existing == nil {
		if err := uc.repo.Create(ctx, verification); err != nil {
			return err
		}
	} else if existing.IsUsed {
		return nil
	} else {
		existing.Code = code
		existing.ExpiresAt = verification.ExpiresAt

		if err := uc.repo.Updates(ctx, existing); err != nil {
			return err
		}
	}

	event := EmailVerificationCreated{
		Email:     email,
		Code:      code,
		Timestamp: time.Now().UTC(),
	}
	event.EventName = event.GetEventName()

	return uc.publisher.PublishEvent(ctx, event)
}
