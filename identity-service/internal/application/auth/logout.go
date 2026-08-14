package auth

import (
	"context"

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

func (uc *Logout) Execute(
	ctx context.Context,
	req LogoutRequest,
) error {
	hashed := uc.manager.Hash(req.RefreshToken)

	if err := uc.repo.Delete(ctx, hashed); err != nil {
		return err
	}

	return nil
}
