package platform

import (
	"github.com/redis/go-redis/v9"
	"github.com/wecrazy/sociomile/backend/internal/config"
)

// OpenRedis opens the Redis client used for cache and rate limiting helpers.
func OpenRedis(cfg config.Config) *redis.Client {
	if cfg.RedisAddr == "" {
		return nil
	}

	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
}
