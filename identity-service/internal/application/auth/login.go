package auth

import (
	"context"

	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/user"
	apperr "identity-service/internal/shared/errors"
)

const refreshTokenExpiry = 7

type Login struct {
	userRepo            user.UserRepository
	passwordHasher      auth.PasswordHasher
	jwtManager          auth.JWTManager
	refreshTokenRepo    auth.RefreshTokenRepository
	refreshTokenManager auth.RefreshTokenManager
}

func NewLogin(
	userRepo user.UserRepository,
	passwordHasher auth.PasswordHasher,
	jwtManager auth.JWTManager,
	refreshTokenRepo auth.RefreshTokenRepository,
	refreshTokenManager auth.RefreshTokenManager,
) *Login {
	return &Login{
		userRepo:            userRepo,
		passwordHasher:      passwordHasher,
		jwtManager:          jwtManager,
		refreshTokenRepo:    refreshTokenRepo,
		refreshTokenManager: refreshTokenManager,
	}
}

func (uc *Login) Execute(
	ctx context.Context,
	input LoginRequest,
) (TokenResponse, error) {
	usr, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return TokenResponse{}, err
	}

	if usr == nil || !uc.passwordHasher.Compare(usr.Password, input.Password) {
		// Deliberately collapsed response prevents user enumeration
		return TokenResponse{}, apperr.ErrUnauthorized
	}

	accessToken, err := uc.jwtManager.Generate(usr.ID.String(), usr.Role)
	if err != nil {
		return TokenResponse{}, err
	}

	refreshToken, err := uc.refreshTokenManager.Generate()
	if err != nil {
		return TokenResponse{}, err
	}

	hashedToken := uc.refreshTokenManager.Hash(refreshToken)

	claims := auth.UserClaims{
		UserID: usr.ID.String(),
		Role:   usr.Role,
	}

	ttlSeconds := int64(refreshTokenExpiry) * 24 * 3600

	err = uc.refreshTokenRepo.Save(ctx, hashedToken, claims, ttlSeconds)
	if err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
