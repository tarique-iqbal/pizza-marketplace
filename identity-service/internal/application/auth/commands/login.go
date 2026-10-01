package commands

import (
	"context"
	"strings"
	"time"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/domain/auth"
	"identity-service/internal/domain/user"
	logobs "identity-service/internal/infrastructure/observability/logger"
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

func (cmd *Login) Execute(
	ctx context.Context,
	input authapp.LoginRequest,
) (authapp.TokenResponse, error) {
	usr, err := cmd.userRepo.FindByEmail(ctx, strings.ToLower(input.Email))
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	if usr == nil || !cmd.passwordHasher.Compare(usr.Password, input.Password) {
		// Deliberately collapsed response prevents user enumeration
		return authapp.TokenResponse{}, apperr.ErrUnauthorized
	}

	accessToken, err := cmd.jwtManager.Generate(usr.ID.String(), usr.Role)
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	refreshToken, err := cmd.refreshTokenManager.Generate()
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	hashedToken := cmd.refreshTokenManager.Hash(refreshToken)

	claims := auth.UserClaims{
		UserID: usr.ID.String(),
		Role:   usr.Role,
	}

	ttlSeconds := int64(refreshTokenExpiry) * 24 * 3600

	err = cmd.refreshTokenRepo.Save(ctx, hashedToken, claims, ttlSeconds)
	if err != nil {
		return authapp.TokenResponse{}, err
	}

	loggedAt := time.Now().UTC()
	usr.LoggedAt = &loggedAt

	if err := cmd.userRepo.Update(ctx, usr); err != nil {
		logobs.FromContext(ctx).Warn("failed to update user's last login", "error", err)
	}

	return authapp.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
