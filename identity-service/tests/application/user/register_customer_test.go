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

func setupRegisterCustomer(t *testing.T) *user.RegisterCustomer {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableEmailVerification, testutil.TableUser)

	_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)
	_ = fixtures.LoadUserFixtures(t, db.DB)

	emailVerificationRepo := persistence.NewEmailVerificationRepository(db.DB)
	codeVerifier := auth.NewEmailVerifier(emailVerificationRepo)
	userRepo := persistence.NewUserRepository(db.DB)
	outboxRepo := persistence.NewOutboxRepository(db.DB)
	hasher := security.NewPasswordHasher()

	return user.NewRegisterCustomer(db.DB, codeVerifier, userRepo, hasher, outboxRepo)
}

func TestRegisterCustomer_Success(t *testing.T) {
	db := testutil.DB(t)
	register := setupRegisterCustomer(t)

	input := user.RegisterCustomerRequest{
		FirstName: "Adam",
		LastName:  "D'Angelo",
		Email:     "adam.dangelo@example.com",
		Password:  "securepassword",
		Code:      "476190", // from fixture
	}

	newUser, err := register.Execute(context.Background(), input)

	assert.Nil(t, err)
	assert.NotNil(t, newUser)
	assert.Equal(t, "Adam", newUser.Name.First)

	outboxEvent := firstOutboxEvent(t, db.DB, newUser.ID, "user.registered")

	var payload struct {
		UserID    string `json:"user_id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
	}
	require.NoError(t, json.Unmarshal(outboxEvent.Payload, &payload))
	assert.Equal(t, newUser.ID.String(), payload.UserID)
	assert.Equal(t, "Adam", payload.FirstName)
	assert.Equal(t, "adam.dangelo@example.com", payload.Email)
}

func TestRegisterCustomer_Failure_EmailVerification(t *testing.T) {
	register := setupRegisterCustomer(t)

	input := user.RegisterCustomerRequest{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "invalid@example.com",
		Password:  "password",
		Code:      "wrong-code", // invalid
	}

	response, err := register.Execute(context.Background(), input)

	assert.Error(t, err)
	assert.Empty(t, response)
}

func TestRegisterCustomer_Failure_DuplicateEmail(t *testing.T) {
	register := setupRegisterCustomer(t)

	input := user.RegisterCustomerRequest{
		FirstName: "Existing",
		LastName:  "User",
		Email:     "existing@example.com", // from fixture
		Password:  "password",
		Code:      "365189",
	}

	response, err := register.Execute(context.Background(), input)

	assert.Error(t, err)
	assert.Empty(t, response)
}
