package redisstore

import (
	"context"

	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/redis/go-redis/v9"
)

type Options struct {
	Addr     string
	Password string
	DB       int
	Debug    bool
}

func ParseFromConfig(c *config.Config) Options {
	return Options{
		Addr:     c.Redis.Addr,
		Password: c.Redis.Password,
		DB:       c.Redis.DB,
		Debug:    c.Debug,
	}
}

func NewConnection(opts Options) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}
