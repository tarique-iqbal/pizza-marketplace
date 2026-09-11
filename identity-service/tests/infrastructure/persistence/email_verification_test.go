package persistence_test

import (
	"context"
	"identity-service/internal/domain/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupEmailVerificationRepo(t *testing.T) auth.EmailVerificationRepository {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableEmailVerification)

	_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)

	return persistence.NewEmailVerificationRepository(db.DB)
}

func TestEmailVerificationRepository_Create(t *testing.T) {
	emVerRepo := setupEmailVerificationRepo(t)

	emailVerification := auth.EmailVerification{
		ID:        testutil.MustNewID(),
		Email:     "adam.dangelo@example.com",
		Code:      "467923",
		IsUsed:    false,
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
		CreatedAt: time.Now().UTC(),
	}

	err := emVerRepo.Create(context.Background(), &emailVerification)

	assert.Nil(t, err)
	assert.NotZero(t, emailVerification.ID)
}

func TestEmailVerificationRepository_Updates(t *testing.T) {
	emVerRepo := setupEmailVerificationRepo(t)

	existing, _ := emVerRepo.FindByEmail(context.Background(), "john.doe@example.com")
	existing.Code = "478326"
	existing.IsUsed = false
	existing.ExpiresAt = time.Now().UTC().Add(15 * time.Minute)

	err := emVerRepo.Updates(context.Background(), existing)

	assert.Nil(t, err)
}

func TestEmailVerificationRepository_FindByEmail(t *testing.T) {
	emVerRepo := setupEmailVerificationRepo(t)

	emVer, err := emVerRepo.FindByEmail(context.Background(), "john.doe@example.com")

	assert.NoError(t, err)
	assert.Equal(t, "john.doe@example.com", emVer.Email)
}
