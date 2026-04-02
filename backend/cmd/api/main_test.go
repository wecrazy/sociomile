package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/config"
	"github.com/wecrazy/sociomile/backend/internal/health"
	"github.com/wecrazy/sociomile/backend/internal/http/handler"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunReturnsConfigError(t *testing.T) {
	t.Setenv("BACKEND_PORT", "not-a-number")
	err := run()
	require.ErrorContains(t, err, "parse BACKEND_PORT")
}

func TestMainHandlesSuccessAndFailure(t *testing.T) {
	originalRunAPI := runAPI
	originalLogAPIError := logAPIError
	originalExitProcess := exitProcess
	t.Cleanup(func() {
		runAPI = originalRunAPI
		logAPIError = originalLogAPIError
		exitProcess = originalExitProcess
	})

	logged := false
	exitCode := 0
	logAPIError = func(msg string, _ ...any) {
		logged = true
		require.Equal(t, "api exited with error", msg)
	}
	exitProcess = func(code int) {
		exitCode = code
	}

	runAPI = func() error { return nil }
	main()
	require.False(t, logged)
	require.Equal(t, 0, exitCode)

	runAPI = func() error { return errors.New("boom") }
	main()
	require.True(t, logged)
	require.Equal(t, 1, exitCode)
}

func TestRunHandlesCLICommands(t *testing.T) {
	originalLoadConfig := loadConfig
	originalOpenDatabase := openDatabase
	originalApplyMigrations := applyMigrations
	originalLoadDemoData := loadDemoData
	originalOpenRedis := openRedis
	originalNewLogger := newLogger
	originalNewRouter := newRouter
	originalListenApp := listenApp
	originalArgs := os.Args
	t.Cleanup(func() {
		loadConfig = originalLoadConfig
		openDatabase = originalOpenDatabase
		applyMigrations = originalApplyMigrations
		loadDemoData = originalLoadDemoData
		openRedis = originalOpenRedis
		newLogger = originalNewLogger
		newRouter = originalNewRouter
		listenApp = originalListenApp
		os.Args = originalArgs
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api-main.db")), &gorm.Config{})
	require.NoError(t, err)

	loadConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", LogLevel: "debug", MySQLDSN: "ignored"}, nil
	}
	openDatabase = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	openRedis = func(_ config.Config) *redis.Client {
		return nil
	}

	migrationCalls := 0
	seedCalls := 0
	applyMigrations = func(_ context.Context, database *gorm.DB, dir string) error {
		migrationCalls++
		require.Equal(t, db, database)
		require.Equal(t, filepath.Join(".", "migrations"), dir)
		return nil
	}
	loadDemoData = func(_ context.Context, database *gorm.DB) error {
		seedCalls++
		require.Equal(t, db, database)
		return nil
	}

	os.Args = []string{"api", "migrate"}
	require.NoError(t, run())
	require.Equal(t, 1, migrationCalls)
	require.Equal(t, 0, seedCalls)

	os.Args = []string{"api", "seed"}
	require.NoError(t, run())
	require.Equal(t, 2, migrationCalls)
	require.Equal(t, 1, seedCalls)

	os.Args = []string{"api", "unknown"}
	err = run()
	require.ErrorContains(t, err, "unknown command: unknown")
	require.Equal(t, 3, migrationCalls)
	require.Equal(t, 1, seedCalls)
}

func TestRunBuildsHandlersAndListensWithoutCLICommand(t *testing.T) {
	originalLoadConfig := loadConfig
	originalOpenDatabase := openDatabase
	originalApplyMigrations := applyMigrations
	originalLoadDemoData := loadDemoData
	originalOpenRedis := openRedis
	originalNewLogger := newLogger
	originalNewRouter := newRouter
	originalListenApp := listenApp
	originalArgs := os.Args
	t.Cleanup(func() {
		loadConfig = originalLoadConfig
		openDatabase = originalOpenDatabase
		applyMigrations = originalApplyMigrations
		loadDemoData = originalLoadDemoData
		openRedis = originalOpenRedis
		newLogger = originalNewLogger
		newRouter = originalNewRouter
		listenApp = originalListenApp
		os.Args = originalArgs
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api-server.db")), &gorm.Config{})
	require.NoError(t, err)

	loadConfig = func() (config.Config, error) {
		return config.Config{
			AppEnv:         "test",
			LogLevel:       "debug",
			MySQLDSN:       "ignored",
			JWTSecret:      "secret",
			AccessTokenTTL: 0,
			Port:           8080,
			SwaggerFile:    "./docs/openapi.yaml",
		}, nil
	}
	openDatabase = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}
	openRedis = func(_ config.Config) *redis.Client {
		return nil
	}
	newLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	routerCalls := 0
	newRouter = func(_ config.Config, _ *slog.Logger, _ *health.Reporter, authHandler *handler.AuthHandler, userHandler *handler.UserHandler, conversationHandler *handler.ConversationHandler, ticketHandler *handler.TicketHandler) *fiber.App {
		routerCalls++
		require.NotNil(t, authHandler)
		require.NotNil(t, userHandler)
		require.NotNil(t, conversationHandler)
		require.NotNil(t, ticketHandler)
		return fiber.New()
	}
	listenApp = func(app *fiber.App, address string) error {
		require.NotNil(t, app)
		require.Equal(t, ":8080", address)
		return errors.New("listen failed")
	}

	os.Args = []string{"api"}
	err = run()
	require.ErrorContains(t, err, "listen failed")
	require.Equal(t, 1, routerCalls)
}

func TestRunEnsuresDemoDataInDevelopment(t *testing.T) {
	originalLoadConfig := loadConfig
	originalOpenDatabase := openDatabase
	originalApplyMigrations := applyMigrations
	originalLoadDemoData := loadDemoData
	originalOpenRedis := openRedis
	originalNewLogger := newLogger
	originalNewRouter := newRouter
	originalListenApp := listenApp
	originalArgs := os.Args
	t.Cleanup(func() {
		loadConfig = originalLoadConfig
		openDatabase = originalOpenDatabase
		applyMigrations = originalApplyMigrations
		loadDemoData = originalLoadDemoData
		openRedis = originalOpenRedis
		newLogger = originalNewLogger
		newRouter = originalNewRouter
		listenApp = originalListenApp
		os.Args = originalArgs
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api-dev-seed.db")), &gorm.Config{})
	require.NoError(t, err)

	loadConfig = func() (config.Config, error) {
		return config.Config{
			AppEnv:         "development",
			LogLevel:       "debug",
			MySQLDSN:       "ignored",
			JWTSecret:      "secret",
			AccessTokenTTL: 0,
			Port:           8080,
			SwaggerFile:    "./docs/openapi.yaml",
		}, nil
	}
	openDatabase = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}
	openRedis = func(_ config.Config) *redis.Client {
		return nil
	}
	newLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	seedCalls := 0
	loadDemoData = func(_ context.Context, database *gorm.DB) error {
		seedCalls++
		require.Equal(t, db, database)
		return nil
	}

	newRouter = func(_ config.Config, _ *slog.Logger, _ *health.Reporter, _ *handler.AuthHandler, _ *handler.UserHandler, _ *handler.ConversationHandler, _ *handler.TicketHandler) *fiber.App {
		return fiber.New()
	}
	listenApp = func(app *fiber.App, address string) error {
		require.NotNil(t, app)
		require.Equal(t, ":8080", address)
		return errors.New("listen failed")
	}

	os.Args = []string{"api"}
	err = run()
	require.ErrorContains(t, err, "listen failed")
	require.Equal(t, 1, seedCalls)
}

func TestRunPropagatesStartupErrors(t *testing.T) {
	originalLoadConfig := loadConfig
	originalOpenDatabase := openDatabase
	originalApplyMigrations := applyMigrations
	originalLoadDemoData := loadDemoData
	originalOpenRedis := openRedis
	originalNewLogger := newLogger
	originalArgs := os.Args
	t.Cleanup(func() {
		loadConfig = originalLoadConfig
		openDatabase = originalOpenDatabase
		applyMigrations = originalApplyMigrations
		loadDemoData = originalLoadDemoData
		openRedis = originalOpenRedis
		newLogger = originalNewLogger
		os.Args = originalArgs
	})

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api-errors.db")), &gorm.Config{})
	require.NoError(t, err)

	loadConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", LogLevel: "debug", MySQLDSN: "ignored"}, nil
	}
	newLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	openRedis = func(_ config.Config) *redis.Client {
		return nil
	}

	openDatabase = func(_ config.Config) (*gorm.DB, error) {
		return nil, errors.New("db failed")
	}
	err = run()
	require.ErrorContains(t, err, "db failed")

	openDatabase = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyMigrations = func(context.Context, *gorm.DB, string) error {
		return errors.New("migration failed")
	}
	err = run()
	require.ErrorContains(t, err, "migration failed")

	applyMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}
	loadDemoData = func(context.Context, *gorm.DB) error {
		return errors.New("seed failed")
	}
	os.Args = []string{"api", "seed"}
	err = run()
	require.ErrorContains(t, err, "seed failed")
}
