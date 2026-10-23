package middleware

import (
	"booking-system/pkg/rdb"
	"booking-system/pkg/response"
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// tokenBucketScript is the canonical Lua script for atomic token bucket operations.
// All language implementations use this exact script for behavioral consistency.
const tokenBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local refill_interval = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

-- Get current state or initialize
local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

-- Initialize if this is the first request
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- Calculate token refill
local time_passed = now - last_refill
local refills = math.floor(time_passed / refill_interval)

if refills > 0 then
    tokens = math.min(capacity, tokens + (refills * refill_rate))
    last_refill = last_refill + (refills * refill_interval)
end

-- Try to consume a token
local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

-- Update state
redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)

-- Expire once an idle bucket would be full again, which matches a fresh one
redis.call('EXPIRE', key, math.ceil(math.ceil(capacity / refill_rate) * refill_interval))

-- Return result: allowed (1 or 0) and remaining tokens
return {allowed, tokens}
`

var tokenBucket = redis.NewScript(tokenBucketScript)

// TokenBucketConfig holds the configuration for a TokenBucket rate limiter.
type TokenBucketConfig struct {
	// Capacity is the maximum number of tokens in the bucket.
	Capacity int

	// RefillRate is the number of tokens added per refill interval.
	RefillRate float64

	// RefillInterval is the time between refills.
	RefillInterval time.Duration

	// Client is the Redis client to use. Required.
	Client *redis.Client
}

// TokenBucket is a Redis-based token bucket rate limiter.
//
// It maintains a fixed capacity of tokens that refill at a constant rate.
// Each request consumes one token. When the bucket is empty, requests are
// denied until tokens refill.
type TokenBucket struct {
	client         *redis.Client
	capacity       int
	refillRate     float64
	refillInterval float64 // in seconds
}

// NewTokenBucket creates a new TokenBucket rate limiter with the given configuration.
func NewTokenBucket(cfg TokenBucketConfig) *TokenBucket {
	if cfg.Capacity == 0 {
		cfg.Capacity = 10
	}
	if cfg.RefillRate == 0 {
		cfg.RefillRate = 1.0
	}
	if cfg.RefillInterval == 0 {
		cfg.RefillInterval = time.Second
	}

	return &TokenBucket{
		client:         cfg.Client,
		capacity:       cfg.Capacity,
		refillRate:     cfg.RefillRate,
		refillInterval: cfg.RefillInterval.Seconds(),
	}
}

// Allow checks if a request should be allowed for the given key.
//
// It atomically checks and updates the token bucket state in Redis.
// Returns whether the request is allowed, the number of remaining tokens,
// and any error encountered.
func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, float64, error) {
	now := float64(time.Now().UnixMicro()) / 1e6

	result, err := tokenBucket.Run(ctx, tb.client, []string{key},
		tb.capacity, tb.refillRate, tb.refillInterval, now).Int64Slice()
	if err != nil {
		return false, 0, fmt.Errorf("token bucket eval failed: %w", err)
	}

	allowed := result[0] == 1
	remaining := float64(result[1])

	return allowed, remaining, nil
}

// NewBucket is the app-wide limiter: bursts of up to 20 requests, refilled at 10 per second.
func NewBucket() *TokenBucket {
	return NewTokenBucket(TokenBucketConfig{
		Client:         rdb.Client,
		Capacity:       20,
		RefillRate:     10,
		RefillInterval: time.Second,
	})
}

// RateLimiterMiddleware gives each keyFn(c) value its own bucket and returns 429 once it's empty.
// If Redis errors, the request is let through rather than blocked.
func RateLimiterMiddleware(limiter *TokenBucket, keyFn func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, remaining, err := limiter.Allow(c.Request.Context(), keyFn(c))
		if err != nil {
			log.Printf("rate limiter skipped: %v", err)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.capacity))
		c.Header("X-RateLimit-Remaining", strconv.FormatFloat(remaining, 'f', 0, 64))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(
			time.Now().Add(time.Duration(limiter.refillInterval*float64(time.Second))).Unix(), 10))

		if !allowed {
			c.Header("Retry-After", "1")
			c.Error(response.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}
