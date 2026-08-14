package auth

import (
	"context"

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

func (uc *RefreshToken) Execute(
	ctx context.Context,
	req RefreshRequest,
) (TokenResponse, error) {
	hashed := uc.manager.Hash(req.RefreshToken)

	claims, err := uc.repo.Find(ctx, hashed)
	if err != nil {
		return TokenResponse{}, err
	}

	accessToken, err := uc.jwtManager.Generate(claims.UserID, claims.Role)
	if err != nil {
		return TokenResponse{}, err
	}

	refreshToken, err := uc.manager.Generate()
	if err != nil {
		return TokenResponse{}, err
	}

	hashedToken := uc.manager.Hash(refreshToken)

	ttlSeconds := int64(refreshTokenExpiry) * 24 * 3600

	err = uc.repo.Save(ctx, hashedToken, claims, ttlSeconds)
	if err != nil {
		return TokenResponse{}, err
	}

	_ = uc.repo.Delete(ctx, hashed)

	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
