package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"identity-service/internal/infrastructure/persistence"
	"identity-service/tests/testutil"
)

func TestOTPRateLimiter_Allow_BlocksWithinCooldown(t *testing.T) {
	rdb := testutil.Redis(t)
	rdb.Flush(t)

	limiter := persistence.NewOTPRateLimiter(rdb.Client, time.Minute)

	allowed, err := limiter.Allow(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = limiter.Allow(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestOTPRateLimiter_Allow_DifferentEmailsIndependent(t *testing.T) {
	rdb := testutil.Redis(t)
	rdb.Flush(t)

	limiter := persistence.NewOTPRateLimiter(rdb.Client, time.Minute)

	allowed, err := limiter.Allow(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = limiter.Allow(context.Background(), "bob@example.com")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestOTPRateLimiter_Allow_AllowsAfterCooldownExpires(t *testing.T) {
	rdb := testutil.Redis(t)
	rdb.Flush(t)

	limiter := persistence.NewOTPRateLimiter(rdb.Client, time.Second)

	allowed, err := limiter.Allow(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.True(t, allowed)

	time.Sleep(2 * time.Second)

	allowed, err = limiter.Allow(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.True(t, allowed)
}
