package persistence

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"identity-service/internal/domain/auth"
)

type otpRateLimiter struct {
	client   *goredis.Client
	cooldown time.Duration
}

func NewOTPRateLimiter(client *goredis.Client, cooldown time.Duration) auth.OTPRateLimiter {
	return &otpRateLimiter{client: client, cooldown: cooldown}
}

func (r *otpRateLimiter) Allow(ctx context.Context, email string) (bool, error) {
	key := "otp_request:" + email

	err := r.client.SetArgs(ctx, key, "1", goredis.SetArgs{
		Mode: "NX",
		TTL:  r.cooldown,
	}).Err()

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, goredis.Nil):
		return false, nil
	default:
		return false, err
	}
}
