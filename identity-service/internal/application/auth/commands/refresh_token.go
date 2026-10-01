package commands

import (
	"context"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
)

type RefreshToken struct {
	jwtManager auth.JWTManager
	repo       auth.RefreshTokenRepository
	manager    auth.RefreshTokenManager
}

func NewRefreshToken(
	jwtManager auth.JWTManager,
	repo auth.RefreshTokenRepository,
	manager auth.RefreshTokenManager,
) *RefreshToken {
	return &RefreshToken{
		jwtManager: jwtManager,
		repo:       repo,
		manager:    manager,
	}
}

func (cmd *RefreshToken) Execute(
	ctx context.Context,
	req authapp.RefreshRequest,
) (authapp.TokenResponse, error) {
	hashed := cmd.manager.Hash(req.RefreshToken)

	claims, err := cmd.repo.Find(ctx, hashed)
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	accessToken, err := cmd.jwtManager.Generate(claims.UserID, claims.Role)
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	refreshToken, err := cmd.manager.Generate()
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	hashedToken := cmd.manager.Hash(refreshToken)

	ttlSeconds := int64(refreshTokenExpiry) * 24 * 3600

	err = cmd.repo.Save(ctx, hashedToken, claims, ttlSeconds)
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	_ = cmd.repo.Delete(ctx, hashed)

	return authapp.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
