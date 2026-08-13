package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/user"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/internal/infrastructure/security"
	"identity-service/internal/shared/event"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
)

var emailOTP *authapp.RequestEmailOTP
var mockPublisher *MockEventPublisher
var repo auth.EmailVerificationRepository

type MockEventPublisher struct {
	PublishedEvents []event.Event
	PublishedRaw    [][]byte
	ShouldFail      bool
}

func (m *MockEventPublisher) PublishEvent(ctx context.Context, e event.Event) error {
	m.PublishedEvents = append(m.PublishedEvents, e)
	if m.ShouldFail {
		return errors.New("mock publish failure")
	}
	return nil
}

func (m *MockEventPublisher) PublishRaw(ctx context.Context, topic string, jsonData []byte) error {
	m.PublishedRaw = append(m.PublishedRaw, jsonData)
	if m.ShouldFail {
		return errors.New("mock raw publish failure")
	}
	return nil
}

func requestEmailOTP(t *testing.T) *authapp.RequestEmailOTP {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableUser, testutil.TableEmailVerification)

	_ = fixtures.LoadUserFixtures(t, db.DB)

	repo = persistence.NewEmailVerificationRepository(db.DB)
	userRepo := persistence.NewUserRepository(db.DB)
	otp := security.NewOTPGenerator()
	mockPublisher = &MockEventPublisher{}

	return authapp.NewRequestEmailOTP(repo, userRepo, otp, mockPublisher)
}

func TestCreateEmailVerification_Success(t *testing.T) {
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

	createdEvent, ok := mockPublisher.PublishedEvents[0].(authapp.EmailVerificationCreated)
	assert.True(t, ok)
	assert.Equal(t, "adam.dangelo@example.com", createdEvent.Email)
	assert.Equal(t, emailVerification.Code, createdEvent.Code)
	assert.Equal(t, "email.verification_created", createdEvent.GetEventName())
	assert.Len(t, mockPublisher.PublishedEvents, 1)
}

func TestCreateEmailVerification_EmailAlreadyRegistered(t *testing.T) {
	emailOTP = requestEmailOTP(t)

	input := authapp.EmailVerificationRequest{
		Email: "john.doe@example.com",
	}

	err := emailOTP.Execute(context.Background(), input)

	assert.ErrorIs(t, err, user.ErrEmailAlreadyExists)
	assert.Empty(t, mockPublisher.PublishedEvents)
}
