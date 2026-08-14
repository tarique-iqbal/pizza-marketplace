package persistence_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"identity-service/internal/domain/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/tests/testutil"
)

func TestRedisConnection(t *testing.T) {
	rdb := testutil.Redis(t)
	err := rdb.Client.Ping(context.Background()).Err()
	require.NoError(t, err)
}

func TestRefreshTokenRepository_SaveAndFind(t *testing.T) {
	ctx := context.Background()

	rdb := testutil.Redis(t)
	rdb.Flush(t)

	repo := persistence.NewRefreshTokenRepository(rdb.Client)

	hashedToken := "test-token"
	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	err := repo.Save(ctx, hashedToken, claims, 60)
	require.NoError(t, err)

	foundClaims, err := repo.Find(ctx, hashedToken)
	require.NoError(t, err)

	assert.Equal(t, claims, foundClaims)
}

func TestRefreshTokenRepository_Find_NotFound(t *testing.T) {
	ctx := context.Background()

	rdb := testutil.Redis(t)
	rdb.Flush(t)

	repo := persistence.NewRefreshTokenRepository(rdb.Client)

	_, err := repo.Find(ctx, "non-existing-token")

	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrRefreshTokenInvalid))
}

func TestRefreshTokenRepository_Delete(t *testing.T) {
	ctx := context.Background()

	rdb := testutil.Redis(t)
	rdb.Flush(t)

	repo := persistence.NewRefreshTokenRepository(rdb.Client)

	hashedToken := "delete-token"
	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	err := repo.Save(ctx, hashedToken, claims, 60)
	require.NoError(t, err)

	err = repo.Delete(ctx, hashedToken)
	require.NoError(t, err)

	_, err = repo.Find(ctx, hashedToken)
	require.Error(t, err)
}

func TestRefreshTokenRepository_TTLExpiry(t *testing.T) {
	ctx := context.Background()

	rdb := testutil.Redis(t)
	rdb.Flush(t)

	repo := persistence.NewRefreshTokenRepository(rdb.Client)

	hashedToken := "ttl-token"
	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	err := repo.Save(ctx, hashedToken, claims, 1)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	_, err = repo.Find(ctx, hashedToken)

	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrRefreshTokenInvalid))
}
