package redisstore

import (
	"context"

	"github.com/SeaCloudHub/backend/domain/pubsub"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	rdb *redis.Client
}

type RedisPubSub struct {
	rps *redis.PubSub
}

func NewRedisClient(rdb *redis.Client) *RedisClient {
	return &RedisClient{rdb: rdb}
}

func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.rdb.Publish(ctx, channel, message).Err()
}

func (r *RedisClient) Subscribe(ctx context.Context, channel string) pubsub.PubSub {
	return &RedisPubSub{rps: r.rdb.Subscribe(ctx, channel)}
}

func (r *RedisPubSub) ReceiveMessage(ctx context.Context) (pubsub.Message, error) {
	msg, err := r.rps.ReceiveMessage(ctx)
	if err != nil {
		return pubsub.Message{}, err
	}
	return pubsub.Message{
		Channel: msg.Channel,
		Payload: msg.Payload,
	}, nil
}

func (r *RedisPubSub) Close() error {
	return r.rps.Close()
}
