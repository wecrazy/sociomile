package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps Redis helpers used by the application.
type Client struct {
	redis *redis.Client
}

// New builds a cache client from a Redis client.
func New(redisClient *redis.Client) *Client {
	return &Client{redis: redisClient}
}

// Enabled reports whether Redis-backed helpers are available.
func (c *Client) Enabled() bool {
	return c != nil && c.redis != nil
}

// Key builds a namespaced cache key from the provided parts.
func (c *Client) Key(parts ...string) string {
	return fmt.Sprintf("sociomile:%s", join(parts...))
}

// GetJSON loads a JSON value into target when the key exists.
func (c *Client) GetJSON(ctx context.Context, key string, target any) (bool, error) {
	if !c.Enabled() {
		return false, nil
	}

	value, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}

		return false, err
	}

	if err := json.Unmarshal(value, target); err != nil {
		return false, err
	}

	return true, nil
}

// SetJSON stores a JSON-encoded value with the given TTL.
func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if !c.Enabled() {
		return nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, payload, ttl).Err()
}

// Version returns the cache version counter for a tenant resource.
func (c *Client) Version(ctx context.Context, tenantID string, resource string) int64 {
	if !c.Enabled() {
		return 1
	}

	key := c.Key("tenant", tenantID, resource, "version")
	version, err := c.redis.Get(ctx, key).Int64()
	if err == redis.Nil {
		_ = c.redis.Set(ctx, key, 1, 0).Err()
		return 1
	}
	if err != nil {
		return 1
	}

	return version
}

// BumpVersion increments the cache version counter for a tenant resource.
func (c *Client) BumpVersion(ctx context.Context, tenantID string, resource string) {
	if !c.Enabled() {
		return
	}

	_ = c.redis.Incr(ctx, c.Key("tenant", tenantID, resource, "version")).Err()
}

// CheckRateLimit increments the key and reports whether the limit is still allowed.
func (c *Client) CheckRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	if !c.Enabled() {
		return true, nil
	}

	count, err := c.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		if err := c.redis.Expire(ctx, key, window).Err(); err != nil {
			return false, err
		}
	}

	return count <= limit, nil
}

func join(parts ...string) string {
	result := ""
	for index, part := range parts {
		if index > 0 {
			result += ":"
		}
		result += part
	}

	return result
}
