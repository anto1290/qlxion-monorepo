package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheRepo provides caching functionality
type CacheRepo struct {
	client *redis.Client
	prefix string
}

// NewCacheRepo creates a new CacheRepo
func NewCacheRepo(client *redis.Client, prefix string) *CacheRepo {
	return &CacheRepo{
		client: client,
		prefix: prefix,
	}
}

// key generates a prefixed key
func (r *CacheRepo) key(key string) string {
	return fmt.Sprintf("%s:%s", r.prefix, key)
}

// Get gets a value from cache
func (r *CacheRepo) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, r.key(key)).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// Set sets a value in cache with TTL
func (r *CacheRepo) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key(key), data, ttl).Err()
}

// Delete deletes a key from cache
func (r *CacheRepo) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.key(key)).Err()
}

// DeletePattern deletes keys matching a pattern
func (r *CacheRepo) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := r.client.Keys(ctx, r.key(pattern)).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Exists checks if a key exists
func (r *CacheRepo) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, r.key(key)).Result()
	return n > 0, err
}

// Increment increments a counter
func (r *CacheRepo) Increment(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, r.key(key)).Result()
}

// SetExpiration sets expiration on a key
func (r *CacheRepo) SetExpiration(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, r.key(key), ttl).Err()
}

// HSet sets a hash field
func (r *CacheRepo) HSet(ctx context.Context, key string, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.HSet(ctx, r.key(key), field, data).Err()
}

// HGet gets a hash field
func (r *CacheRepo) HGet(ctx context.Context, key string, field string, dest interface{}) error {
	data, err := r.client.HGet(ctx, r.key(key), field).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// HDelete deletes a hash field
func (r *CacheRepo) HDelete(ctx context.Context, key string, field string) error {
	return r.client.HDel(ctx, r.key(key), field).Err()
}

// AddToBlacklist adds a token to the blacklist
func (r *CacheRepo) AddToBlacklist(ctx context.Context, tokenID string, ttl time.Duration) error {
	return r.client.Set(ctx, r.key("blacklist:"+tokenID), "1", ttl).Err()
}

// IsBlacklisted checks if a token is blacklisted
func (r *CacheRepo) IsBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	exists, err := r.client.Exists(ctx, r.key("blacklist:"+tokenID)).Result()
	return exists > 0, err
}
