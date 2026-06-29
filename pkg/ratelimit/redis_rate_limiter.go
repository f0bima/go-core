package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements distributed rate limiting using Redis.
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter.
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

// Allow checks if a request is allowed using sliding window counter.
// Returns: allowed, current count, limit, reset time
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, int64, time.Time, error) {
	now := time.Now()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now.Unix()/int64(window.Seconds()))

	pipe := rl.client.TxPipeline()
	
	// Increment counter
	incr := pipe.Incr(ctx, windowKey)
	// Set expiry
	pipe.Expire(ctx, windowKey, window*2) // 2x window for safety

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, 0, limit, time.Time{}, fmt.Errorf("redis rate limit: %w", err)
	}

	count := incr.Val()
	resetTime := now.Add(window)

	allowed := count <= limit

	return allowed, count, limit, resetTime, nil
}

// AllowWithRetry attempts rate limit check with retry on failure.
// Falls back to allow if Redis is unavailable (fail-open strategy).
func (rl *RedisRateLimiter) AllowWithRetry(ctx context.Context, key string, limit int64, window time.Duration, maxRetries int) (bool, int64, int64, time.Time, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		allowed, count, limit, resetTime, err := rl.Allow(ctx, key, limit, window)
		if err == nil {
			return allowed, count, limit, resetTime, nil
		}

		lastErr = err

		// Wait before retry
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return false, 0, limit, time.Time{}, ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
	}

	// Fail-open: allow request if Redis is unavailable
	return true, 0, limit, time.Time{}, fmt.Errorf("redis unavailable, allowing: %w", lastErr)
}

// GetRemaining returns remaining requests in current window.
func (rl *RedisRateLimiter) GetRemaining(ctx context.Context, key string, limit int64, window time.Duration) (int64, error) {
	now := time.Now()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now.Unix()/int64(window.Seconds()))

	count, err := rl.client.Get(ctx, windowKey).Int64()
	if err == redis.Nil {
		return limit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("redis get remaining: %w", err)
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// Reset clears rate limit counters for a key.
func (rl *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	pattern := fmt.Sprintf("ratelimit:%s:*", key)
	
	iter := rl.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := rl.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("redis reset: %w", err)
		}
	}

	return iter.Err()
}
