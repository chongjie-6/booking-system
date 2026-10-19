package redis

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
		// Fail fast when redis is down so the fail-open rate limiter adds no latency
		// (default dial + command retries add ~1.7s to every request)
		DialTimeout:   200 * time.Millisecond,
		DialerRetries: 1,
		MaxRetries:    -1,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis unreachable, rate limiting disabled until it connects: %v", err)
	}

	return rdb
}
