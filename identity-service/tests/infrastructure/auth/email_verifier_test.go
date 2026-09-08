package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"identity-service/internal/domain/auth"
	authinfra "identity-service/internal/infrastructure/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/tests/infrastructure/db/fixtures"
	"identity-service/tests/testutil"
)

func setupCodeVerification(t *testing.T) auth.EmailVerifier {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableEmailVerification)

	_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)

	repo := persistence.NewEmailVerificationRepository(db.DB)

	return authinfra.NewEmailVerifier(repo)
}

func TestVerify_Success(t *testing.T) {
	svc := setupCodeVerification(t)

	err := svc.Verify(context.Background(), "alice@example.com", "347578")
	assert.NoError(t, err)
}

func TestVerify_CodeMismatch(t *testing.T) {
	svc := setupCodeVerification(t)

	err := svc.Verify(context.Background(), "alice@example.com", "010101")
	assert.ErrorIs(t, err, auth.ErrCodeInvalid)
}

func TestVerify_CodeMismatch_IncrementsAttemptCount(t *testing.T) {
	db := testutil.DB(t)
	db.TruncateTables(t, testutil.TableEmailVerification)
	_ = fixtures.LoadEmailVerificationFixtures(t, db.DB)

	repo := persistence.NewEmailVerificationRepository(db.DB)
	svc := authinfra.NewEmailVerifier(repo)

	err := svc.Verify(context.Background(), "alice@example.com", "010101")
	assert.ErrorIs(t, err, auth.ErrCodeInvalid)

	ev, err := repo.FindByEmail(context.Background(), "alice@example.com")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, ev.AttemptCount)
}

func TestVerify_TooManyAttempts(t *testing.T) {
	svc := setupCodeVerification(t)

	for i := 0; i < 3; i++ {
		err := svc.Verify(context.Background(), "alice@example.com", "010101")
		assert.ErrorIs(t, err, auth.ErrCodeInvalid)
	}

	err := svc.Verify(context.Background(), "alice@example.com", "010101")
	assert.ErrorIs(t, err, auth.ErrTooManyAttempts)

	// Locked out even with the correct code, and no information leak about correctness.
	err = svc.Verify(context.Background(), "alice@example.com", "347578")
	assert.ErrorIs(t, err, auth.ErrTooManyAttempts)
}

func TestVerify_AlreadyUsed(t *testing.T) {
	svc := setupCodeVerification(t)

	err := svc.Verify(context.Background(), "already.used@example.com", "137468")
	assert.ErrorIs(t, err, auth.ErrCodeUsed)
}

func TestVerify_Expired(t *testing.T) {
	svc := setupCodeVerification(t)

	err := svc.Verify(context.Background(), "expired@example.com", "743802")
	assert.ErrorIs(t, err, auth.ErrCodeExpired)
}

func TestVerify_CodeNotIssued(t *testing.T) {
	svc := setupCodeVerification(t)

	err := svc.Verify(context.Background(), "not.found@example.com", "578578")
	assert.ErrorIs(t, err, auth.ErrCodeNotIssued)
}
