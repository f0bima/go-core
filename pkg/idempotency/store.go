package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyRecord represents a stored idempotency key and its response.
type IdempotencyRecord struct {
	Key        string
	StatusCode int
	Body       []byte
	CreatedAt  time.Time
}

// Store manages idempotency keys and their responses.
type Store struct {
	client *redis.Client
	ttl    time.Duration
}

// NewStore creates a new idempotency store with Redis backend.
func NewStore(client *redis.Client, ttl time.Duration) *Store {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default TTL
	}
	return &Store{
		client: client,
		ttl:    ttl,
	}
}

// CheckIdempotencyKey checks if a key has already been processed.
// Returns (found, record, error).
func (s *Store) CheckIdempotencyKey(ctx context.Context, key string) (bool, *IdempotencyRecord, error) {
	data, err := s.client.Get(ctx, s.idempotencyKey(key)).Result()
	if err == redis.Nil {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("redis get idempotency: %w", err)
	}

	var record IdempotencyRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return false, nil, fmt.Errorf("unmarshal idempotency record: %w", err)
	}

	return true, &record, nil
}

// SaveIdempotencyResponse stores the response for an idempotency key.
func (s *Store) SaveIdempotencyResponse(ctx context.Context, key string, statusCode int, body []byte) error {
	record := IdempotencyRecord{
		Key:        key,
		StatusCode: statusCode,
		Body:       body,
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal idempotency record: %w", err)
	}

	err = s.client.Set(ctx, s.idempotencyKey(key), data, s.ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set idempotency: %w", err)
	}

	return nil
}

// DeleteIdempotencyKey removes an idempotency key.
func (s *Store) DeleteIdempotencyKey(ctx context.Context, key string) error {
	err := s.client.Del(ctx, s.idempotencyKey(key)).Err()
	if err != nil {
		return fmt.Errorf("redis delete idempotency: %w", err)
	}
	return nil
}

// CleanupExpiredKeys removes expired idempotency keys.
// This is optional as Redis TTL handles automatic cleanup.
func (s *Store) CleanupExpiredKeys(ctx context.Context) error {
	// Redis TTL automatically expires keys, so manual cleanup is not needed
	// This method is provided for monitoring/logging purposes
	return nil
}

// GetRemainingTTL returns remaining TTL for an idempotency key.
func (s *Store) GetRemainingTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := s.client.TTL(ctx, s.idempotencyKey(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("redis ttl: %w", err)
	}
	return ttl, nil
}

// idempotencyKey generates the Redis key for an idempotency key.
func (s *Store) idempotencyKey(key string) string {
	return fmt.Sprintf("idempotency:%s", key)
}

// ValidateIdempotencyKey validates the format of an idempotency key.
func ValidateIdempotencyKey(key string) bool {
	// Key should be non-empty and reasonable length
	if len(key) == 0 || len(key) > 255 {
		return false
	}
	return true
}
