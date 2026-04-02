package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wecrazy/sociomile/backend/internal/cache"
	"github.com/wecrazy/sociomile/backend/internal/config"
	"github.com/wecrazy/sociomile/backend/internal/events"
	"github.com/wecrazy/sociomile/backend/internal/health"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"github.com/wecrazy/sociomile/backend/internal/platform"
	"github.com/wecrazy/sociomile/backend/internal/repository"
	"gorm.io/gorm"
)

type eventPublisher interface {
	Publish(ctx context.Context, routingKey string, payload []byte) error
	Close() error
}

type outboxStore interface {
	ListPendingOutbox(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	MarkOutboxFailed(ctx context.Context, eventID string) error
	MarkOutboxPublished(ctx context.Context, eventID string) error
}

var (
	runWorker             = run
	logWorkerError        = func(msg string, args ...any) { slog.Error(msg, args...) }
	exitProcess           = os.Exit
	loadWorkerConfig      = config.Load
	newWorkerLogger       = platform.NewLogger
	openWorkerDB          = platform.OpenDatabase
	openWorkerRedis       = platform.OpenRedis
	applyWorkerMigrations = platform.ApplyMigrations
	notifyContext         = signal.NotifyContext
	newOutboxStore        = func(db *gorm.DB) outboxStore { return repository.NewStore(db) }
	newWorkerCache        = cache.New
	newWorkerTicker       = func(duration time.Duration) (<-chan time.Time, func()) {
		ticker := time.NewTicker(duration)
		return ticker.C, ticker.Stop
	}
	recordHeartbeat = health.RecordWorkerHeartbeat
	waitPublisher   = func(ctx context.Context, logger *slog.Logger, url string) (eventPublisher, error) {
		return waitForPublisher(ctx, logger, url)
	}
	newPublisher = events.NewPublisher
	after        = time.After
)

func main() {
	if err := runWorker(); err != nil {
		logWorkerError("worker exited with error", slog.Any("error", err))
		exitProcess(1)
	}
}

func run() error {
	cfg, err := loadWorkerConfig()
	if err != nil {
		return err
	}

	logger := newWorkerLogger(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(logger)

	db, err := openWorkerDB(cfg)
	if err != nil {
		return err
	}

	ctx, stop := notifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := applyWorkerMigrations(ctx, db, filepath.Join(".", "migrations")); err != nil {
		return err
	}

	cacheClient := newWorkerCache(openWorkerRedis(cfg))
	store := newOutboxStore(db)
	publisher, err := waitPublisher(ctx, logger, cfg.RabbitMQURL)
	if err != nil {
		return err
	}
	defer publisher.Close()

	tickerChannel, stopTicker := newWorkerTicker(5 * time.Second)
	defer stopTicker()

	if err := recordHeartbeat(ctx, cacheClient); err != nil {
		logger.Warn("record worker heartbeat", slog.Any("error", err))
	}

	logger.Info("outbox worker started")
	for {
		select {
		case <-ctx.Done():
			logger.Info("outbox worker stopped")
			return nil
		case <-tickerChannel:
			if err := recordHeartbeat(ctx, cacheClient); err != nil {
				logger.Warn("record worker heartbeat", slog.Any("error", err))
			}

			pending, err := store.ListPendingOutbox(ctx, 50)
			if err != nil {
				logger.Error("load outbox events", slog.Any("error", err))
				continue
			}

			for _, event := range pending {
				if err := publisher.Publish(ctx, event.RoutingKey, []byte(event.Payload)); err != nil {
					logger.Error("publish outbox event",
						slog.String("event_id", event.ID),
						slog.String("routing_key", event.RoutingKey),
						slog.Any("error", err),
					)
					_ = store.MarkOutboxFailed(ctx, event.ID)
					continue
				}

				if err := store.MarkOutboxPublished(ctx, event.ID); err != nil {
					logger.Error("mark outbox published", slog.String("event_id", event.ID), slog.Any("error", err))
					continue
				}

				logger.Info("published outbox event",
					slog.String("event_id", event.ID),
					slog.String("event_type", event.EventType),
				)
			}
		}
	}
}

func waitForPublisher(ctx context.Context, logger *slog.Logger, url string) (*events.Publisher, error) {
	var waitingSince time.Time
	retryCount := 0

	for {
		publisher, err := newPublisher(url)
		if err == nil {
			if !waitingSince.IsZero() {
				logger.Info("rabbitmq became available",
					slog.Duration("wait_time", time.Since(waitingSince).Round(time.Second)),
					slog.Int("retry_count", retryCount),
				)
			}

			return publisher, nil
		}

		retryCount++
		if waitingSince.IsZero() {
			waitingSince = time.Now()
			logger.Info("waiting for rabbitmq during startup", slog.Any("error", err))
		} else if time.Since(waitingSince) >= 30*time.Second && retryCount%15 == 0 {
			logger.Warn("still waiting for rabbitmq",
				slog.Duration("wait_time", time.Since(waitingSince).Round(time.Second)),
				slog.Int("retry_count", retryCount),
				slog.Any("error", err),
			)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-after(2 * time.Second):
		}
	}
}
