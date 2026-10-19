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

	err := rdb.Ping(ctx).Err()

	if err != nil {
		rdb.Close()
		log.Printf("Failed to instantiate redis")
		return nil
	}

	return rdb
}
