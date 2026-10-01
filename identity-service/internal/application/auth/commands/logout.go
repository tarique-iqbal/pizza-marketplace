package commands

import (
	"context"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
)

type Logout struct {
	repo    auth.RefreshTokenRepository
	manager auth.RefreshTokenManager
}

func NewLogout(
	repo auth.RefreshTokenRepository,
	manager auth.RefreshTokenManager,
) *Logout {
	return &Logout{
		repo:    repo,
		manager: manager,
	}
}

func (cmd *Logout) Execute(
	ctx context.Context,
	req authapp.LogoutRequest,
) error {
	hashed := cmd.manager.Hash(req.RefreshToken)

	if err := cmd.repo.Delete(ctx, hashed); err != nil {
		return err
	}

	return nil
}
