package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"APP_ENV", "BACKEND_PORT", "MYSQL_DSN", "REDIS_ADDR", "REDIS_PASSWORD", "RABBITMQ_URL", "JWT_SECRET", "ACCESS_TOKEN_TTL", "LOG_LEVEL", "SWAGGER_FILE"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "development", cfg.AppEnv)
	require.Equal(t, 8080, cfg.Port)
	require.Equal(t, "localhost:16379", cfg.RedisAddr)
	require.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.RabbitMQURL)
	require.Equal(t, 15*time.Minute, cfg.AccessTokenTTL)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "./docs/openapi.yaml", cfg.SwaggerFile)
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("BACKEND_PORT", "9090")
	t.Setenv("MYSQL_DSN", "mysql-dsn")
	t.Setenv("REDIS_ADDR", "redis:6380")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("RABBITMQ_URL", "amqp://rabbit")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("ACCESS_TOKEN_TTL", "30m")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("SWAGGER_FILE", "/tmp/openapi.yaml")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "test", cfg.AppEnv)
	require.Equal(t, 9090, cfg.Port)
	require.Equal(t, "mysql-dsn", cfg.MySQLDSN)
	require.Equal(t, "redis:6380", cfg.RedisAddr)
	require.Equal(t, "secret", cfg.RedisPassword)
	require.Equal(t, "amqp://rabbit", cfg.RabbitMQURL)
	require.Equal(t, "jwt-secret", cfg.JWTSecret)
	require.Equal(t, 30*time.Minute, cfg.AccessTokenTTL)
	require.Equal(t, "warn", cfg.LogLevel)
	require.Equal(t, "/tmp/openapi.yaml", cfg.SwaggerFile)
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Setenv("BACKEND_PORT", "not-a-number")
	_, err := Load()
	require.ErrorContains(t, err, "parse BACKEND_PORT")

	t.Setenv("BACKEND_PORT", "8080")
	t.Setenv("ACCESS_TOKEN_TTL", "not-a-duration")
	_, err = Load()
	require.ErrorContains(t, err, "parse ACCESS_TOKEN_TTL")
}
