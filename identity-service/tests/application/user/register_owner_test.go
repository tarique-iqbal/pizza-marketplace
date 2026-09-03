package user_test

import (
	"context"
	"encoding/json"
	"identity-service/internal/application/user"
	"identity-service/internal/infrastructure/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/internal/infrastructure/security"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRegisterOwner(t *testing.T) *user.RegisterOwner {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableEmailVerification, testutil.TableUser)

	_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)
	_ = fixtures.LoadUserFixtures(t, db.DB)

	emailVerificationRepo := persistence.NewEmailVerificationRepository(db.DB)
	codeVerifier := auth.NewEmailVerifier(emailVerificationRepo)
	userRepo := persistence.NewUserRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	hasher := security.NewPasswordHasher()

	return user.NewRegisterOwner(db.DB, codeVerifier, hasher, userRepo, outboxRepo)
}

func TestRegisterOwner_Success(t *testing.T) {
	db := testutil.DB(t)
	registerOwner := setupRegisterOwner(t)

	input := user.RegisterOwnerRequest{
		FirstName:    "Sophie",
		LastName:     "Müller",
		Email:        "sophie.mueller@example.com",
		Password:     "securepassword",
		Code:         "365189",
		BusinessName: "Domino's Pizza",
		VATNumber:    "DE987654321",
	}

	newUser, err := registerOwner.Execute(context.Background(), input)

	assert.Nil(t, err)
	assert.NotNil(t, newUser)
	assert.Equal(t, "Sophie", newUser.Name.First)

	outboxEvent := firstOutboxEvent(t, db.DB, newUser.ID, "user.registered")

	var payload struct {
		UserID    string `json:"user_id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
	}
	require.NoError(t, json.Unmarshal(outboxEvent.Payload, &payload))
	assert.Equal(t, newUser.ID.String(), payload.UserID)
	assert.Equal(t, "Sophie", payload.FirstName)
	assert.Equal(t, "sophie.mueller@example.com", payload.Email)
}

func TestRegisterOwner_Failure_EmailVerification(t *testing.T) {
	registerOwner := setupRegisterOwner(t)

	input := user.RegisterOwnerRequest{
		FirstName:    "John",
		LastName:     "Doe",
		Email:        "invalid@example.com",
		Password:     "password",
		Code:         "wrong-code", // invalid
		BusinessName: "Test Biz",
		VATNumber:    "DE111654321",
	}

	res, err := registerOwner.Execute(context.Background(), input)

	assert.Error(t, err)
	assert.Empty(t, res)
}

func TestRegisterOwner_Failure_DuplicateEmail(t *testing.T) {
	registerOwner := setupRegisterOwner(t)

	input := user.RegisterOwnerRequest{
		FirstName:    "Existing",
		LastName:     "User",
		Email:        "existing@example.com", // from fixture
		Password:     "password",
		Code:         "365189",
		BusinessName: "Biz",
		VATNumber:    "DE222654321",
	}

	res, err := registerOwner.Execute(context.Background(), input)

	assert.Error(t, err)
	assert.Empty(t, res)
}
