package testutil

import (
	"context"
	"os"
	"sync"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"identity-service/internal/infrastructure/redis"
)

type TestRedis struct {
	Client *goredis.Client
}

var (
	redisOnce sync.Once
	rdb       *TestRedis
)

func Redis(t *testing.T) *TestRedis {
	redisOnce.Do(func() {
		redisStore, err := redis.NewRedis(redis.Config{
			Addr: os.Getenv("REDIS_ADDR"),
			DB:   1,
		})
		if err != nil {
			panic(err)
		}

		rdb = &TestRedis{
			Client: redisStore.Client,
		}
	})

	require.NotNil(t, rdb)

	return rdb
}

func (trc *TestRedis) Flush(t *testing.T) {
	err := trc.Client.FlushDB(context.Background()).Err()
	require.NoError(t, err)
}
