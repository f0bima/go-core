package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds configuration for retry logic.
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      bool
	RetryableFn func(error) bool // Custom function to determine if error is retryable
}

// DefaultRetryConfig returns sensible defaults for retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Jitter:     true,
		RetryableFn: func(err error) bool {
			return err != nil // Retry all errors by default
		},
	}
}

// Retry executes the given function with exponential backoff retry logic.
// Returns the error from the last attempt if all retries fail.
func Retry(ctx context.Context, config *RetryConfig, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		lastErr = fn(ctx)
		
		// Success - return immediately
		if lastErr == nil {
			return nil
		}

		// Check if error is retryable
		if config.RetryableFn != nil && !config.RetryableFn(lastErr) {
			return lastErr
		}

		// Last attempt - return error
		if attempt == config.MaxRetries {
			return lastErr
		}

		// Calculate delay with exponential backoff
		delay := config.BaseDelay * time.Duration(math.Pow(2, float64(attempt)))
		
		// Cap at max delay
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// Add jitter to prevent thundering herd
		if config.Jitter {
			delay = delay + time.Duration(rand.Int63n(int64(delay)/2))
		}

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// RetryWithResult executes the given function with retry and returns the result.
func RetryWithResult[T any](ctx context.Context, config *RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		result, err := fn(ctx)
		
		// Success - return result
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableFn != nil && !config.RetryableFn(err) {
			return zero, err
		}

		// Last attempt - return error
		if attempt == config.MaxRetries {
			return zero, lastErr
		}

		// Calculate delay
		delay := config.BaseDelay * time.Duration(math.Pow(2, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		if config.Jitter {
			delay = delay + time.Duration(rand.Int63n(int64(delay)/2))
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, lastErr
}
