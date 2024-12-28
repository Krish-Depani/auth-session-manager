package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
}

func GetRedisClient(addr, password string, db int) (*RedisClient, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return &RedisClient{Client: redisClient}, nil
}

func (r *RedisClient) SetSession(ctx context.Context, token string, userID uint, expiration time.Duration) error {
	return r.Client.Set(ctx, token, userID, expiration).Err()
}

func (r *RedisClient) GetSession(ctx context.Context, token string) (uint, error) {
	val, err := r.Client.Get(ctx, token).Uint64()

	return uint(val), err
}

func (r *RedisClient) DeleteSession(ctx context.Context, token string) error {
	return r.Client.Del(ctx, token).Err()
}
