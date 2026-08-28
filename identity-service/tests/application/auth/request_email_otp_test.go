package auth_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/outbox"
	"identity-service/internal/domain/user"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/internal/infrastructure/security"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
)

var emailOTP *authapp.RequestEmailOTP
var repo auth.EmailVerificationRepository

func requestEmailOTP(t *testing.T) *authapp.RequestEmailOTP {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableUser, testutil.TableEmailVerification, testutil.TableOutboxEvent)

	_ = fixtures.LoadUserFixtures(t, db.DB)

	repo = persistence.NewEmailVerificationRepository(db.DB)
	userRepo := persistence.NewUserRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	otp := security.NewOTPGenerator()

	return authapp.NewRequestEmailOTP(db.DB, repo, userRepo, otp, outboxRepo)
}

func TestCreateEmailVerification_Success(t *testing.T) {
	db := testutil.DB(t)
	emailOTP = requestEmailOTP(t)

	input := authapp.EmailVerificationRequest{
		Email: "adam.dangelo@example.com",
	}

	err := emailOTP.Execute(context.Background(), input)
	emailVerification, _ := repo.FindByEmail(context.Background(), input.Email)
	diff := emailVerification.ExpiresAt.Sub(emailVerification.CreatedAt)

	assert.Nil(t, err)
	assert.NotNil(t, emailVerification)
	assert.Equal(t, "adam.dangelo@example.com", emailVerification.Email)
	assert.InDelta(t, 15, diff.Minutes(), 0.001, "Delta threshold exceeded")

	var outboxEvent outbox.OutboxEvent
	require.NoError(t, db.DB.Where("event_name = ?", "email.verification_created").First(&outboxEvent).Error)

	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(outboxEvent.Payload, &payload))
	assert.Equal(t, "adam.dangelo@example.com", payload.Email)
	assert.Equal(t, emailVerification.Code, payload.Code)
}

func TestCreateEmailVerification_EmailAlreadyRegistered(t *testing.T) {
	db := testutil.DB(t)
	emailOTP = requestEmailOTP(t)

	input := authapp.EmailVerificationRequest{
		Email: "john.doe@example.com",
	}

	err := emailOTP.Execute(context.Background(), input)

	assert.ErrorIs(t, err, user.ErrEmailAlreadyExists)

	var count int64
	require.NoError(t, db.DB.Model(&outbox.OutboxEvent{}).
		Where("event_name = ?", "email.verification_created").Count(&count).Error)
	assert.Zero(t, count)
}
