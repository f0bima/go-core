package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements a thread-safe token bucket rate limiter.
type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket rate limiter.
// maxTokens: maximum tokens in the bucket (burst size)
// refillRate: tokens added per second
func NewTokenBucket(maxTokens float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed (consumes 1 token).
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN checks if N requests are allowed (consumes N tokens).
func (tb *TokenBucket) AllowN(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}
	return false
}

// Wait blocks until a token is available or context is cancelled.
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}

		// Calculate wait time
		tb.mu.Lock()
		waitTime := time.Duration(float64(time.Second) / tb.refillRate)
		tb.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Try again
		}
	}
}

// refill adds tokens based on elapsed time.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate

	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	tb.lastRefill = now
}

// Tokens returns current available tokens (for monitoring).
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}
