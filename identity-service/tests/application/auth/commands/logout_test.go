package commands_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/application/auth/commands"
	"identity-service/internal/domain/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/internal/infrastructure/security"
	"identity-service/tests/testutil"
)

func setupLogout(t *testing.T) (
	*commands.Logout,
	auth.RefreshTokenManager,
	auth.RefreshTokenRepository,
) {
	rdb := testutil.Redis(t)
	rdb.Flush(t)

	repo := persistence.NewRefreshTokenRepository(rdb.Client)
	manager := security.NewRefreshTokenManager()

	logout := commands.NewLogout(repo, manager)

	return logout, manager, repo
}

func TestLogout_Success(t *testing.T) {
	ctx := context.Background()

	logout, manager, repo := setupLogout(t)

	rawToken, _ := manager.Generate()
	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	err := repo.Save(ctx, hashed, claims, ttl)
	require.NoError(t, err)

	err = logout.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: rawToken,
	})

	require.NoError(t, err)

	_, err = repo.Find(ctx, hashed)
	require.Error(t, err)
}

func TestLogout_Idempotent(t *testing.T) {
	ctx := context.Background()

	logout, manager, repo := setupLogout(t)

	rawToken, _ := manager.Generate()
	hashed := manager.Hash(rawToken)

	claims := auth.UserClaims{
		UserID: testutil.MustNewIDString(),
		Role:   "owner",
	}

	ttl := int64(7) * 24 * 3600
	_ = repo.Save(ctx, hashed, claims, ttl)

	// First call
	err := logout.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: rawToken,
	})
	require.NoError(t, err)

	// Second call (same token)
	err = logout.Execute(ctx, authapp.RefreshRequest{
		RefreshToken: rawToken,
	})
	require.NoError(t, err)
}
