package auth

import (
	"context"
	"encoding/json"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/outbox"
	"identity-service/internal/domain/user"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const accessTokenExpiry = 15

type RequestEmailOTP struct {
	db          *gorm.DB
	repo        auth.EmailVerificationRepository
	userRepo    user.UserRepository
	otp         auth.OTPGenerator
	outboxRepo  outbox.OutboxRepository
	rateLimiter auth.OTPRateLimiter
}

func NewRequestEmailOTP(
	db *gorm.DB,
	repo auth.EmailVerificationRepository,
	userRepo user.UserRepository,
	otp auth.OTPGenerator,
	outboxRepo outbox.OutboxRepository,
	rateLimiter auth.OTPRateLimiter,
) *RequestEmailOTP {
	return &RequestEmailOTP{
		db:          db,
		repo:        repo,
		userRepo:    userRepo,
		otp:         otp,
		outboxRepo:  outboxRepo,
		rateLimiter: rateLimiter,
	}
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

	allowed, err := uc.rateLimiter.Allow(ctx, email)
	if err != nil {
		return err
	}
	if !allowed {
		return auth.ErrTooManyRequests
	}

	code, err := uc.otp.Generate(true)
	if err != nil {
		return err
	}

	verificationID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	expiry := time.Duration(accessTokenExpiry) * time.Minute
	verification := &auth.EmailVerification{
		ID:        verificationID,
		Email:     email,
		Code:      code,
		IsUsed:    false,
		ExpiresAt: time.Now().UTC().Add(expiry),
	}

	existing, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}

	aggregateID := verification.ID
	if existing != nil {
		aggregateID = existing.ID
	}

	emailVerificationCreated := EmailVerificationCreated{
		Email:      email,
		Code:       code,
		OccurredAt: time.Now().UTC(),
	}
	emailVerificationCreated.EventName = emailVerificationCreated.GetEventName()

	payload, err := json.Marshal(emailVerificationCreated)
	if err != nil {
		return err
	}

	return uc.db.Transaction(func(tx *gorm.DB) error {
		if existing == nil {
			if err := uc.repo.WithTx(tx).Create(ctx, verification); err != nil {
				return err
			}
		} else {
			existing.Code = code
			existing.ExpiresAt = verification.ExpiresAt
			existing.AttemptCount = 0
			existing.IsUsed = false

			if err := uc.repo.WithTx(tx).Updates(ctx, existing); err != nil {
				return err
			}
		}

		outboxEvent := outbox.NewOutboxEvent(aggregateID, outbox.EventEmailVerificationCreated, payload)
		return uc.outboxRepo.WithTx(tx).Create(ctx, &outboxEvent)
	})
}
