package redis

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis unreachable, rate limiting disabled until it connects: %v", err)
	}

	return rdb
}
