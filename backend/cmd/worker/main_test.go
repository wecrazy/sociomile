package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/cache"
	"github.com/wecrazy/sociomile/backend/internal/config"
	"github.com/wecrazy/sociomile/backend/internal/events"
	"github.com/wecrazy/sociomile/backend/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunReturnsConfigError(t *testing.T) {
	t.Setenv("BACKEND_PORT", "not-a-number")
	err := run()
	require.ErrorContains(t, err, "parse BACKEND_PORT")
}

func TestMainHandlesSuccessAndFailure(t *testing.T) {
	originalRunWorker := runWorker
	originalLogWorkerError := logWorkerError
	originalExitProcess := exitProcess
	t.Cleanup(func() {
		runWorker = originalRunWorker
		logWorkerError = originalLogWorkerError
		exitProcess = originalExitProcess
	})

	logged := false
	exitCode := 0
	logWorkerError = func(msg string, _ ...any) {
		logged = true
		require.Equal(t, "worker exited with error", msg)
	}
	exitProcess = func(code int) {
		exitCode = code
	}

	runWorker = func() error { return nil }
	main()
	require.False(t, logged)
	require.Equal(t, 0, exitCode)

	runWorker = func() error { return errors.New("boom") }
	main()
	require.True(t, logged)
	require.Equal(t, 1, exitCode)
}

func TestWaitForPublisherHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	publisher, err := waitForPublisher(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), "amqp://%zz")
	require.Nil(t, publisher)
	require.ErrorIs(t, err, context.Canceled)
}

func TestWaitForPublisherRetriesThenSucceeds(t *testing.T) {
	originalNewPublisher := newPublisher
	originalAfter := after
	t.Cleanup(func() {
		newPublisher = originalNewPublisher
		after = originalAfter
	})

	attempts := 0
	newPublisher = func(url string) (*events.Publisher, error) {
		attempts++
		require.Equal(t, "amqp://broker", url)
		if attempts == 1 {
			return nil, errors.New("broker warming up")
		}

		return &events.Publisher{}, nil
	}
	after = func(_ time.Duration) <-chan time.Time {
		channel := make(chan time.Time, 1)
		channel <- time.Now()
		return channel
	}

	publisher, err := waitForPublisher(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), "amqp://broker")
	require.NoError(t, err)
	require.NotNil(t, publisher)
	require.Equal(t, 2, attempts)
}

func TestWaitForPublisherLogsStartupWaitAsInfo(t *testing.T) {
	originalNewPublisher := newPublisher
	originalAfter := after
	t.Cleanup(func() {
		newPublisher = originalNewPublisher
		after = originalAfter
	})

	attempts := 0
	records := make([]slog.Record, 0, 2)
	logger := slog.New(&captureHandler{records: &records})

	newPublisher = func(url string) (*events.Publisher, error) {
		attempts++
		require.Equal(t, "amqp://broker", url)
		if attempts == 1 {
			return nil, errors.New("broker warming up")
		}

		return &events.Publisher{}, nil
	}
	after = func(_ time.Duration) <-chan time.Time {
		channel := make(chan time.Time, 1)
		channel <- time.Now()
		return channel
	}

	publisher, err := waitForPublisher(context.Background(), logger, "amqp://broker")
	require.NoError(t, err)
	require.NotNil(t, publisher)
	require.Len(t, records, 2)
	require.Equal(t, slog.LevelInfo, records[0].Level)
	require.Equal(t, "waiting for rabbitmq during startup", records[0].Message)
	require.Equal(t, slog.LevelInfo, records[1].Level)
	require.Equal(t, "rabbitmq became available", records[1].Message)
}

type captureHandler struct {
	mx      sync.Mutex
	records *[]slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, record slog.Record) error {
	h.mx.Lock()
	defer h.mx.Unlock()
	clone := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		clone.AddAttrs(attr)
		return true
	})
	*h.records = append(*h.records, clone)
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(string) slog.Handler {
	return h
}

func TestRunStopsCleanlyWhenContextIsAlreadyCancelled(t *testing.T) {
	originalLoadWorkerConfig := loadWorkerConfig
	originalNewWorkerLogger := newWorkerLogger
	originalOpenWorkerDB := openWorkerDB
	originalApplyWorkerMigrations := applyWorkerMigrations
	originalNotifyContext := notifyContext
	originalNewOutboxStore := newOutboxStore
	originalWaitPublisher := waitPublisher
	originalNewWorkerTicker := newWorkerTicker
	t.Cleanup(func() {
		loadWorkerConfig = originalLoadWorkerConfig
		newWorkerLogger = originalNewWorkerLogger
		openWorkerDB = originalOpenWorkerDB
		applyWorkerMigrations = originalApplyWorkerMigrations
		notifyContext = originalNotifyContext
		newOutboxStore = originalNewOutboxStore
		waitPublisher = originalWaitPublisher
		newWorkerTicker = originalNewWorkerTicker
	})

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/worker.db"), &gorm.Config{})
	require.NoError(t, err)

	loadWorkerConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", LogLevel: "debug", MySQLDSN: "ignored", RabbitMQURL: "amqp://broker"}, nil
	}
	newWorkerLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	openWorkerDB = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyWorkerMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}
	newOutboxStore = func(_ *gorm.DB) outboxStore {
		return &stubOutboxStore{}
	}
	notifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		cancel()
		return ctx, func() {}
	}
	newWorkerTicker = func(_ time.Duration) (<-chan time.Time, func()) {
		channel := make(chan time.Time)
		return channel, func() { close(channel) }
	}
	waitPublisher = func(_ context.Context, _ *slog.Logger, url string) (eventPublisher, error) {
		require.Equal(t, "amqp://broker", url)
		return &events.Publisher{}, nil
	}

	require.NoError(t, run())
}

func TestRunPropagatesStartupErrors(t *testing.T) {
	originalLoadWorkerConfig := loadWorkerConfig
	originalNewWorkerLogger := newWorkerLogger
	originalOpenWorkerDB := openWorkerDB
	originalApplyWorkerMigrations := applyWorkerMigrations
	originalWaitPublisher := waitPublisher
	t.Cleanup(func() {
		loadWorkerConfig = originalLoadWorkerConfig
		newWorkerLogger = originalNewWorkerLogger
		openWorkerDB = originalOpenWorkerDB
		applyWorkerMigrations = originalApplyWorkerMigrations
		waitPublisher = originalWaitPublisher
	})

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/worker-errors.db"), &gorm.Config{})
	require.NoError(t, err)

	loadWorkerConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", LogLevel: "debug", MySQLDSN: "ignored", RabbitMQURL: "amqp://broker"}, nil
	}
	newWorkerLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	openWorkerDB = func(_ config.Config) (*gorm.DB, error) {
		return nil, errors.New("db failed")
	}
	err = run()
	require.ErrorContains(t, err, "db failed")

	openWorkerDB = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyWorkerMigrations = func(context.Context, *gorm.DB, string) error {
		return errors.New("migration failed")
	}
	err = run()
	require.ErrorContains(t, err, "migration failed")

	applyWorkerMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}
	waitPublisher = func(context.Context, *slog.Logger, string) (eventPublisher, error) {
		return nil, errors.New("publisher failed")
	}
	err = run()
	require.ErrorContains(t, err, "publisher failed")
}

func TestRunContinuesAfterOutboxLoadError(t *testing.T) {
	originalLoadWorkerConfig := loadWorkerConfig
	originalNewWorkerLogger := newWorkerLogger
	originalOpenWorkerDB := openWorkerDB
	originalApplyWorkerMigrations := applyWorkerMigrations
	originalNotifyContext := notifyContext
	originalNewOutboxStore := newOutboxStore
	originalWaitPublisher := waitPublisher
	originalNewWorkerTicker := newWorkerTicker
	t.Cleanup(func() {
		loadWorkerConfig = originalLoadWorkerConfig
		newWorkerLogger = originalNewWorkerLogger
		openWorkerDB = originalOpenWorkerDB
		applyWorkerMigrations = originalApplyWorkerMigrations
		notifyContext = originalNotifyContext
		newOutboxStore = originalNewOutboxStore
		waitPublisher = originalWaitPublisher
		newWorkerTicker = originalNewWorkerTicker
	})

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/worker-loop-error.db"), &gorm.Config{})
	require.NoError(t, err)

	loadWorkerConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", LogLevel: "debug", MySQLDSN: "ignored", RabbitMQURL: "amqp://broker"}, nil
	}
	newWorkerLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	openWorkerDB = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyWorkerMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	notifyContext = func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return ctx, cancel
	}
	tick := make(chan time.Time, 1)
	newWorkerTicker = func(_ time.Duration) (<-chan time.Time, func()) {
		return tick, func() {}
	}
	newOutboxStore = func(_ *gorm.DB) outboxStore {
		return &stubOutboxStore{
			listPendingOutboxFn: func(context.Context, int) ([]model.OutboxEvent, error) {
				cancel()
				return nil, errors.New("load failed")
			},
		}
	}
	waitPublisher = func(context.Context, *slog.Logger, string) (eventPublisher, error) {
		return &stubPublisher{}, nil
	}

	tick <- time.Now()
	require.NoError(t, run())
}

func TestRunProcessesOutboxEvents(t *testing.T) {
	originalLoadWorkerConfig := loadWorkerConfig
	originalNewWorkerLogger := newWorkerLogger
	originalOpenWorkerDB := openWorkerDB
	originalApplyWorkerMigrations := applyWorkerMigrations
	originalNotifyContext := notifyContext
	originalNewOutboxStore := newOutboxStore
	originalWaitPublisher := waitPublisher
	originalNewWorkerTicker := newWorkerTicker
	t.Cleanup(func() {
		loadWorkerConfig = originalLoadWorkerConfig
		newWorkerLogger = originalNewWorkerLogger
		openWorkerDB = originalOpenWorkerDB
		applyWorkerMigrations = originalApplyWorkerMigrations
		notifyContext = originalNotifyContext
		newOutboxStore = originalNewOutboxStore
		waitPublisher = originalWaitPublisher
		newWorkerTicker = originalNewWorkerTicker
	})

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/worker-loop.db"), &gorm.Config{})
	require.NoError(t, err)

	loadWorkerConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", LogLevel: "debug", MySQLDSN: "ignored", RabbitMQURL: "amqp://broker"}, nil
	}
	newWorkerLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	openWorkerDB = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyWorkerMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	notifyContext = func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return ctx, cancel
	}
	tick := make(chan time.Time, 1)
	newWorkerTicker = func(_ time.Duration) (<-chan time.Time, func()) {
		return tick, func() {}
	}

	store := &stubOutboxStore{}
	store.listPendingOutboxFn = func(context.Context, int) ([]model.OutboxEvent, error) {
		return []model.OutboxEvent{
			{BaseModel: model.BaseModel{ID: "event-1"}, EventType: "failed", RoutingKey: "failed", Payload: "{}"},
			{BaseModel: model.BaseModel{ID: "event-2"}, EventType: "mark-error", RoutingKey: "mark-error", Payload: "{}"},
			{BaseModel: model.BaseModel{ID: "event-3"}, EventType: "published", RoutingKey: "published", Payload: "{}"},
		}, nil
	}
	store.markOutboxPublishedFn = func(_ context.Context, eventID string) error {
		store.published = append(store.published, eventID)
		if eventID == "event-2" {
			return errors.New("mark failed")
		}
		if eventID == "event-3" {
			cancel()
		}
		return nil
	}
	store.markOutboxFailedFn = func(_ context.Context, eventID string) error {
		store.failed = append(store.failed, eventID)
		return nil
	}
	newOutboxStore = func(_ *gorm.DB) outboxStore {
		return store
	}

	publisher := &stubPublisher{
		publishFn: func(_ context.Context, routingKey string, _ []byte) error {
			publisherCalls := routingKey
			_ = publisherCalls
			if routingKey == "failed" {
				return errors.New("publish failed")
			}
			return nil
		},
	}
	waitPublisher = func(context.Context, *slog.Logger, string) (eventPublisher, error) {
		return publisher, nil
	}

	tick <- time.Now()
	require.NoError(t, run())
	require.Equal(t, []string{"event-1"}, store.failed)
	require.Equal(t, []string{"event-2", "event-3"}, store.published)
	require.True(t, publisher.closed)
}

func TestRunRecordsWorkerHeartbeatOnStartupAndTick(t *testing.T) {
	originalLoadWorkerConfig := loadWorkerConfig
	originalNewWorkerLogger := newWorkerLogger
	originalOpenWorkerDB := openWorkerDB
	originalApplyWorkerMigrations := applyWorkerMigrations
	originalNotifyContext := notifyContext
	originalNewOutboxStore := newOutboxStore
	originalWaitPublisher := waitPublisher
	originalNewWorkerTicker := newWorkerTicker
	originalRecordHeartbeat := recordHeartbeat
	t.Cleanup(func() {
		loadWorkerConfig = originalLoadWorkerConfig
		newWorkerLogger = originalNewWorkerLogger
		openWorkerDB = originalOpenWorkerDB
		applyWorkerMigrations = originalApplyWorkerMigrations
		notifyContext = originalNotifyContext
		newOutboxStore = originalNewOutboxStore
		waitPublisher = originalWaitPublisher
		newWorkerTicker = originalNewWorkerTicker
		recordHeartbeat = originalRecordHeartbeat
	})

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/worker-heartbeat.db"), &gorm.Config{})
	require.NoError(t, err)

	loadWorkerConfig = func() (config.Config, error) {
		return config.Config{AppEnv: "test", LogLevel: "debug", MySQLDSN: "ignored", RabbitMQURL: "amqp://broker"}, nil
	}
	newWorkerLogger = func(_ string, _ string) *slog.Logger {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	openWorkerDB = func(_ config.Config) (*gorm.DB, error) {
		return db, nil
	}
	applyWorkerMigrations = func(context.Context, *gorm.DB, string) error {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	notifyContext = func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return ctx, cancel
	}
	tick := make(chan time.Time, 1)
	newWorkerTicker = func(_ time.Duration) (<-chan time.Time, func()) {
		return tick, func() {}
	}
	newOutboxStore = func(_ *gorm.DB) outboxStore {
		return &stubOutboxStore{
			listPendingOutboxFn: func(context.Context, int) ([]model.OutboxEvent, error) {
				cancel()
				return nil, nil
			},
		}
	}
	waitPublisher = func(context.Context, *slog.Logger, string) (eventPublisher, error) {
		return &stubPublisher{}, nil
	}

	heartbeatCalls := 0
	recordHeartbeat = func(context.Context, *cache.Client) error {
		heartbeatCalls++
		return nil
	}

	tick <- time.Now()
	require.NoError(t, run())
	require.Equal(t, 2, heartbeatCalls)
}

type stubPublisher struct {
	publishFn func(ctx context.Context, routingKey string, payload []byte) error
	closed    bool
}

func (p *stubPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	if p.publishFn != nil {
		return p.publishFn(ctx, routingKey, payload)
	}

	return nil
}

func (p *stubPublisher) Close() error {
	p.closed = true
	return nil
}

type stubOutboxStore struct {
	listPendingOutboxFn   func(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	markOutboxFailedFn    func(ctx context.Context, eventID string) error
	markOutboxPublishedFn func(ctx context.Context, eventID string) error
	failed                []string
	published             []string
}

func (s *stubOutboxStore) ListPendingOutbox(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	if s.listPendingOutboxFn != nil {
		return s.listPendingOutboxFn(ctx, limit)
	}

	return nil, nil
}

func (s *stubOutboxStore) MarkOutboxFailed(ctx context.Context, eventID string) error {
	if s.markOutboxFailedFn != nil {
		return s.markOutboxFailedFn(ctx, eventID)
	}

	s.failed = append(s.failed, eventID)
	return nil
}

func (s *stubOutboxStore) MarkOutboxPublished(ctx context.Context, eventID string) error {
	if s.markOutboxPublishedFn != nil {
		return s.markOutboxPublishedFn(ctx, eventID)
	}

	s.published = append(s.published, eventID)
	return nil
}
