package container

import (
	"os"
	"time"

	authcmd "identity-service/internal/application/auth/commands"
	"identity-service/internal/application/health"
	usercmd "identity-service/internal/application/user/commands"
	userqry "identity-service/internal/application/user/queries"
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
	otpRateLimiter := persistence.NewOTPRateLimiter(redisStore.Client, 60*time.Second)
	emailVerificationRepo := persistence.NewEmailVerificationRepository(base.Postgres.DB)
	userRepo := persistence.NewUserRepository(base.Postgres.DB)

	codeVerifier := authinfra.NewEmailVerifier(emailVerificationRepo)

	// user
	registerCustomer := usercmd.NewRegisterCustomer(
		base.Postgres.DB, codeVerifier, userRepo, hasher, base.OutboxRepo,
	)
	registerOwner := usercmd.NewRegisterOwner(
		base.Postgres.DB, codeVerifier, hasher, userRepo, base.OutboxRepo,
	)
	findByID := userqry.NewFindByID(userRepo)
	userHandler := http.NewUserHandler(registerCustomer, registerOwner, findByID)

	// auth
	login := authcmd.NewLogin(userRepo, hasher, jwtManager, refreshTokenRepo, refreshTokenManager)
	emailOTP := authcmd.NewRequestEmailOTP(
		base.Postgres.DB, emailVerificationRepo, userRepo, otp, base.OutboxRepo, otpRateLimiter,
	)
	refreshToken := authcmd.NewRefreshToken(jwtManager, refreshTokenRepo, refreshTokenManager)
	logout := authcmd.NewLogout(refreshTokenRepo, refreshTokenManager)
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
