// services/backlog-service/internal/adapters/cache/redis.go

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/ubmm/backlog-service/internal/config"
)

// RedisAdapter implements the cache provider interface
type RedisAdapter struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisAdapter creates a new Redis adapter
func NewRedisAdapter(cfg config.CacheConfig, logger *zap.Logger) (*RedisAdapter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		
		// Connection pool settings
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		
		// Connection timeouts
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		
		// TLS configuration if needed
		TLSConfig: cfg.TLSEnabled,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisAdapter{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (a *RedisAdapter) Close() error {
	return a.client.Close()
}

// Get retrieves a value from cache
func (a *RedisAdapter) Get(ctx context.Context, key string) (interface{}, error) {
	// Add namespace prefix to key
	key = a.prefixKey(key)
	
	// Get from Redis
	val, err := a.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	// Unmarshal value
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return result, nil
}

// Set stores a value in cache with expiration
func (a *RedisAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Add namespace prefix to key
	key = a.prefixKey(key)
	
	// Marshal value
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	// Set in Redis
	err = a.client.Set(ctx, key, jsonBytes, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set in Redis: %w", err)
	}

	return nil
}

// Delete removes a value from cache
func (a *RedisAdapter) Delete(ctx context.Context, key string) error {
	// Add namespace prefix to key
	key = a.prefixKey(key)
	
	// Delete from Redis
	err := a.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	return nil
}

// FlushAll removes all values from cache
func (a *RedisAdapter) FlushAll(ctx context.Context) error {
	err := a.client.FlushAll(ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to flush Redis: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache
func (a *RedisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	// Add namespace prefix to key
	key = a.prefixKey(key)
	
	// Check existence
	result, err := a.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence in Redis: %w", err)
	}

	return result > 0, nil
}

// Increment increments a counter value
func (a *RedisAdapter) Increment(ctx context.Context, key string, value int64) (int64, error) {
	// Add namespace prefix to key
	key = a.prefixKey(key)
	
	// Increment counter
	result, err := a.client.IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment in Redis: %w", err)
	}

	return result, nil
}

// GetTTL gets the time-to-live of a key
func (a *RedisAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	// Add namespace prefix to key
	key = a.prefixKey(key)
	
	// Get TTL
	result, err := a.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL from Redis: %w", err)
	}

	return result, nil
}

// Keys gets all keys matching a pattern
func (a *RedisAdapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	// Add namespace prefix to pattern
	pattern = a.prefixKey(pattern)
	
	// Get keys
	result, err := a.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys from Redis: %w", err)
	}

	// Remove namespace prefix from keys
	for i, key := range result {
		result[i] = a.unprefixKey(key)
	}

	return result, nil
}

// DeleteByPattern deletes all keys matching a pattern
func (a *RedisAdapter) DeleteByPattern(ctx context.Context, pattern string) error {
	// Add namespace prefix to pattern
	pattern = a.prefixKey(pattern)
	
	// Get keys
	keys, err := a.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys from Redis: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete keys
	err = a.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete keys from Redis: %w", err)
	}

	return nil
}

// Helper methods

// Key prefix for namespacing
const keyPrefix = "ubmm:backlog:"

func (a *RedisAdapter) prefixKey(key string) string {
	return keyPrefix + key
}

func (a *RedisAdapter) unprefixKey(key string) string {
	return key[len(keyPrefix):]
}
