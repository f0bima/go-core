package cache

import (
	"context"
	"fmt"
)

// Provider defines the interface for cache operations.
type Provider interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl int64) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetTTL(ctx context.Context, key string, ttl int64) error
	GetTTL(ctx context.Context, key string) (int64, error)
	Ping(ctx context.Context) error
	Close() error
}

// ErrCacheMiss is returned when a key is not found in cache.
var ErrCacheMiss = fmt.Errorf("cache miss")

// Config holds Redis connection configuration.
type Config struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}
