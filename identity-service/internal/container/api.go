package container

import (
	"os"

	authapp "identity-service/internal/application/auth"
	"identity-service/internal/application/health"
	"identity-service/internal/application/user"
	authinfra "identity-service/internal/infrastructure/auth"
	"identity-service/internal/infrastructure/persistence"
	"identity-service/internal/infrastructure/redis"
	"identity-service/internal/infrastructure/security"
	"identity-service/internal/interfaces/http"
	"identity-service/internal/interfaces/http/middlewares"
)

type APIContainer struct {
	*Shared
	UserHandler   *http.UserHandler
	AuthHandler   *http.AuthHandler
	HealthHandler *http.HealthHandler
	Middleware    *middlewares.Middleware
}

func NewAPIContainer() (*APIContainer, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	cfg := redis.Config{
		Addr: os.Getenv("REDIS_ADDR"),
	}

	base, err := NewShared()
	if err != nil {
		return nil, err
	}

	redisStore, err := redis.NewRedis(cfg)
	if err != nil {
		return nil, err
	}

	hasher := security.NewPasswordHasher()
	jwtManager := security.NewJWTManager(jwtSecret)
	refreshTokenManager := security.NewRefreshTokenManager()
	middleware := middlewares.NewMiddleware(jwtManager)
	otp := security.NewOTPGenerator()

	refreshTokenRepo := persistence.NewRefreshTokenRepository(redisStore.Client)
	emailVerificationRepo := persistence.NewEmailVerificationRepository(base.Postgres.DB)
	userRepo := persistence.NewUserRepository(base.Postgres.DB)

	codeVerifier := authinfra.NewEmailVerifier(emailVerificationRepo)

	// user
	registerCustomer := user.NewRegisterCustomer(codeVerifier, userRepo, hasher, base.Publisher)
	registerOwner := user.NewRegisterOwner(
		base.Postgres.DB, codeVerifier, hasher, userRepo, base.OutboxRepo, base.Publisher,
	)
	findByID := user.NewFindByID(userRepo)
	userHandler := http.NewUserHandler(registerCustomer, registerOwner, findByID)

	// auth
	login := authapp.NewLogin(userRepo, hasher, jwtManager, refreshTokenRepo, refreshTokenManager)
	emailOTP := authapp.NewRequestEmailOTP(emailVerificationRepo, userRepo, otp, base.Publisher)
	refreshToken := authapp.NewRefreshToken(jwtManager, refreshTokenRepo, refreshTokenManager)
	logout := authapp.NewLogout(refreshTokenRepo, refreshTokenManager)
	authHandler := http.NewAuthHandler(login, emailOTP, refreshToken, logout)

	// health
	readiness := health.NewReadiness(base.Postgres, redisStore, base.Publisher)
	healthHandler := http.NewHealthHandler(readiness)

	return &APIContainer{
		Shared:        base,
		UserHandler:   userHandler,
		AuthHandler:   authHandler,
		HealthHandler: healthHandler,
		Middleware:    middleware,
	}, nil
}
