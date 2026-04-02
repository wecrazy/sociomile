package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the runtime configuration for backend processes.
type Config struct {
	AppEnv         string
	Port           int
	MySQLDSN       string
	RedisAddr      string
	RedisPassword  string
	RabbitMQURL    string
	JWTSecret      string
	AccessTokenTTL time.Duration
	LogLevel       string
	SwaggerFile    string
}

// Load reads runtime configuration from environment variables with local defaults.
func Load() (Config, error) {
	port, err := getEnvInt("BACKEND_PORT", 8080)
	if err != nil {
		return Config{}, fmt.Errorf("parse BACKEND_PORT: %w", err)
	}

	ttl, err := getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, fmt.Errorf("parse ACCESS_TOKEN_TTL: %w", err)
	}

	cfg := Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           port,
		MySQLDSN:       getEnv("MYSQL_DSN", "sociomile:sociomile@tcp(localhost:13306)/sociomile?parseTime=true&multiStatements=true"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:16379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RabbitMQURL:    getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me"),
		AccessTokenTTL: ttl,
		LogLevel:       getEnv("LOG_LEVEL", "debug"),
		SwaggerFile:    getEnv("SWAGGER_FILE", "./docs/openapi.yaml"),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	value := getEnv(key, "")
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := getEnv(key, "")
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}
