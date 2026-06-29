package cache

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
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

// RedisCache implements Provider using Redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance with connection pool.
func NewRedisCache(cfg *Config) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  5 * time.Second,
	})

	return &RedisCache{client: rdb}
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", fmt.Errorf("redis get: %w", err)
	}
	return val, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value string, ttl int64) error {
	var expiration time.Duration
	if ttl > 0 {
		expiration = time.Duration(ttl) * time.Second
	}

	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis delete: %w", err)
	}
	return nil
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	val, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return val > 0, nil
}

func (r *RedisCache) SetTTL(ctx context.Context, key string, ttl int64) error {
	expiration := time.Duration(ttl) * time.Second
	err := r.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		return fmt.Errorf("redis set ttl: %w", err)
	}
	return nil
}

func (r *RedisCache) GetTTL(ctx context.Context, key string) (int64, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis ttl: %w", err)
	}

	if ttl == -1 {
		return -1, nil // No expiration
	}
	if ttl < 0 {
		return -2, nil // Key doesn't exist
	}

	return int64(ttl.Seconds()), nil
}

func (r *RedisCache) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

func (r *RedisCache) Close() error {
	err := r.client.Close()
	if err != nil {
		return fmt.Errorf("redis close: %w", err)
	}
	return nil
}

// HealthCheck performs a health check on Redis connection.
func (r *RedisCache) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return r.Ping(ctx)
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// InitCache initializes Redis cache and verifies connection.
func InitCache() Provider {
	cfg := &Config{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       parseInt(getEnv("REDIS_DB", "0")),
		PoolSize: parseInt(getEnv("REDIS_POOL_SIZE", "10")),
	}

	cache := NewRedisCache(cfg)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cache.Ping(ctx); err != nil {
		slog.Warn("Redis connection failed, cache will be disabled", "error", err)
		return nil
	}

	slog.Info("Redis cache initialized successfully", "addr", cfg.Addr)
	return cache
}
