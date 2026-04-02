package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"
	"github.com/wecrazy/sociomile/backend/internal/cache"
	"github.com/wecrazy/sociomile/backend/internal/config"
	"github.com/wecrazy/sociomile/backend/internal/health"
	"github.com/wecrazy/sociomile/backend/internal/http/handler"
	"github.com/wecrazy/sociomile/backend/internal/http/router"
	"github.com/wecrazy/sociomile/backend/internal/platform"
	"github.com/wecrazy/sociomile/backend/internal/repository"
	"github.com/wecrazy/sociomile/backend/internal/service"
	"github.com/wecrazy/sociomile/backend/seeds"
)

var (
	runAPI          = run
	logAPIError     = func(msg string, args ...any) { slog.Error(msg, args...) }
	exitProcess     = os.Exit
	loadConfig      = config.Load
	openDatabase    = platform.OpenDatabase
	applyMigrations = platform.ApplyMigrations
	loadDemoData    = seeds.LoadDemoData
	openRedis       = platform.OpenRedis
	newLogger       = platform.NewLogger
	newRouter       = router.New
	listenApp       = func(app *fiber.App, address string) error { return app.Listen(address) }
)

func main() {
	if err := runAPI(); err != nil {
		logAPIError("api exited with error", slog.Any("error", err))
		exitProcess(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	logger := newLogger(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(logger)

	db, err := openDatabase(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	migrationsDir := filepath.Join(".", "migrations")
	if err := applyMigrations(ctx, db, migrationsDir); err != nil {
		return err
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			logger.Info("migrations applied", slog.String("dir", migrationsDir))
			return nil
		case "seed":
			if err := loadDemoData(ctx, db); err != nil {
				return err
			}
			logger.Info("seed data loaded")
			return nil
		default:
			return fmt.Errorf("unknown command: %s", os.Args[1])
		}
	}

	if cfg.AppEnv == "development" {
		if err := loadDemoData(ctx, db); err != nil {
			return err
		}
		logger.Info("demo data ensured")
	}

	redisClient := openRedis(cfg)
	cacheClient := cache.New(redisClient)
	store := repository.NewStore(db)

	authService := service.NewAuthService(store, cfg.JWTSecret, cfg.AccessTokenTTL)
	userService := service.NewUserService(store)
	conversationService := service.NewConversationService(store, cacheClient)
	ticketService := service.NewTicketService(store, cacheClient)
	healthReporter := health.NewReporter(cacheClient)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	conversationHandler := handler.NewConversationHandler(conversationService)
	ticketHandler := handler.NewTicketHandler(ticketService)

	app := newRouter(cfg, logger, healthReporter, authHandler, userHandler, conversationHandler, ticketHandler)
	logger.Info("starting api", slog.Int("port", cfg.Port))
	return listenApp(app, fmt.Sprintf(":%d", cfg.Port))
}
