package rdb

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

// Connect doesn't exit if Redis is down, because the rate limiter lets requests through without it.
// The short dial timeout and disabled retries stop a missing Redis from slowing down every request.
func Connect(ctx context.Context) {
	Client = redis.NewClient(&redis.Options{
		Addr:          os.Getenv("REDIS_ADDRESS"),
		Password:      os.Getenv("REDIS_PASSWORD"),
		DB:            0,
		DialTimeout:   200 * time.Millisecond,
		DialerRetries: 1,
		MaxRetries:    -1,
	})

	if err := Client.Ping(ctx).Err(); err != nil {
		log.Printf("redis unreachable, rate limiting disabled until it connects: %v", err)
	}
}
