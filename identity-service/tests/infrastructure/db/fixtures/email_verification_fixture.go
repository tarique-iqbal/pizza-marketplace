package fixtures

import (
	"identity-service/internal/domain/auth"
	"identity-service/tests/testutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func LoadEmailVerificationFixtures(t *testing.T, db *gorm.DB) error {
	emailVerification := []auth.EmailVerification{
		{
			ID:        testutil.MustNewID(),
			Email:     "john.doe@example.com",
			Code:      "135864",
			IsUsed:    false,
			ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        testutil.MustNewID(),
			Email:     "adam.dangelo@example.com",
			Code:      "476190",
			IsUsed:    false,
			ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        testutil.MustNewID(),
			Email:     "alice@example.com",
			Code:      "347578",
			IsUsed:    false,
			ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        testutil.MustNewID(),
			Email:     "sophie.mueller@example.com",
			Code:      "365189",
			IsUsed:    false,
			ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        testutil.MustNewID(),
			Email:     "already.used@example.com",
			Code:      "137468",
			IsUsed:    true,
			ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
		{
			ID:        testutil.MustNewID(),
			Email:     "expired@example.com",
			Code:      "743802",
			IsUsed:    false,
			ExpiresAt: time.Now().UTC().Add(-1 * time.Minute),
			CreatedAt: time.Now().UTC(),
		},
	}

	for _, u := range emailVerification {
		err := db.Create(&u).Error
		require.NoError(t, err)
	}

	return nil
}
