package auth_test

import (
	"context"
	"errors"
	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/internal/infrastructure/security"
	"identity-service/tests/testutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var refresher *authapp.RefreshToken

func setupRefreshToken(t *testing.T) (
	*authapp.RefreshToken,
	auth.RefreshTokenManager,
	auth.RefreshTokenRepository,
) {
	rdb := testutil.Redis(t)
	rdb.Flush(t)

	jwt := security.NewJWTManager("TestSecretKey")
	repo := persistence.NewRefreshTokenRepository(rdb.Client)
	manager := security.NewRefreshTokenManager()

	refresher = authapp.NewRefreshToken(jwt, repo, manager)

	return refresher, manager, repo
}

func TestRefreshToken_Success(t *testing.T) {
	ctx := context.Background()

	refresher, manager, repo := setupRefreshToken(t)

	rawToken, err := manager.Generate()
	require.NoError(t, err)

	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	err = repo.Save(ctx, hashed, claims, ttl)
	require.NoError(t, err)

	response, err := refresher.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	require.NoError(t, err)

	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
	assert.NotEqual(t, rawToken, response.RefreshToken)
}

func TestRefreshToken_Rotation_InvalidatesOldToken(t *testing.T) {
	ctx := context.Background()

	refresher, manager, repo := setupRefreshToken(t)

	rawToken, _ := manager.Generate()
	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	_ = repo.Save(ctx, hashed, claims, ttl)

	_, err := refresher.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: rawToken,
	})
	require.NoError(t, err)

	_, err = refresher.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrRefreshTokenInvalid))
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	ctx := context.Background()

	refresher, _, _ := setupRefreshToken(t)

	_, err := refresher.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: "invalid-token",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrRefreshTokenInvalid))
}

func TestRefreshToken_ExpiredToken(t *testing.T) {
	ctx := context.Background()

	refresher, manager, repo := setupRefreshToken(t)

	rawToken, _ := manager.Generate()
	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	_ = repo.Save(ctx, hashed, claims, 1)

	time.Sleep(2 * time.Second)

	_, err := refresher.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, auth.ErrRefreshTokenInvalid))
}
