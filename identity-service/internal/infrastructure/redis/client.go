package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	Client *goredis.Client
}

type Config struct {
	Addr     string
	Password string
	DB       int
}

func NewRedis(cfg Config) (*Redis, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Redis{
		Client: client,
	}, nil
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}
