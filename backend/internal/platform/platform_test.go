package platform

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wecrazy/sociomile/backend/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewLoggerAndParseLevel(t *testing.T) {
	require.NotNil(t, NewLogger("development", "debug"))
	require.NotNil(t, NewLogger("production", "warn"))
	require.Equal(t, slog.LevelDebug, parseLevel("debug"))
	require.Equal(t, slog.LevelWarn, parseLevel("warn"))
	require.Equal(t, slog.LevelError, parseLevel("error"))
	require.Equal(t, slog.LevelInfo, parseLevel("anything-else"))
}

func TestOpenRedis(t *testing.T) {
	require.Nil(t, OpenRedis(config.Config{}))

	client := OpenRedis(config.Config{RedisAddr: "localhost:6379", RedisPassword: "secret"})
	require.NotNil(t, client)
	require.Equal(t, "localhost:6379", client.Options().Addr)
	require.Equal(t, "secret", client.Options().Password)
	require.NoError(t, client.Close())
}

func TestApplyMigrations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migrations.db")), &gorm.Config{})
	require.NoError(t, err)

	migrationsDir := t.TempDir()
	require.NoError(t, writeFile(filepath.Join(migrationsDir, "001_create_widgets.up.sql"), []byte("CREATE TABLE widgets (id INTEGER PRIMARY KEY);")))
	require.NoError(t, writeFile(filepath.Join(migrationsDir, "002_create_gadgets.up.sql"), []byte("CREATE TABLE gadgets (id INTEGER PRIMARY KEY);")))
	require.NoError(t, writeFile(filepath.Join(migrationsDir, "003_blank.up.sql"), []byte("   \n")))
	require.NoError(t, writeFile(filepath.Join(migrationsDir, "README.txt"), []byte("ignored")))

	require.NoError(t, ApplyMigrations(context.Background(), db, migrationsDir))
	require.NoError(t, ApplyMigrations(context.Background(), db, migrationsDir))

	var count int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM schema_migrations").Scan(&count).Error)
	require.EqualValues(t, 2, count)

	var widgets int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'widgets'").Scan(&widgets).Error)
	require.EqualValues(t, 1, widgets)

	var gadgets int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'gadgets'").Scan(&gadgets).Error)
	require.EqualValues(t, 1, gadgets)
}

func TestApplyMigrationsReturnsErrors(t *testing.T) {
	t.Run("closed database", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "closed.db")), &gorm.Config{})
		require.NoError(t, err)

		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())

		err = ApplyMigrations(context.Background(), db, t.TempDir())
		require.ErrorContains(t, err, "create schema_migrations")
	})

	t.Run("missing directory", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "missing-dir.db")), &gorm.Config{})
		require.NoError(t, err)

		err = ApplyMigrations(context.Background(), db, filepath.Join(t.TempDir(), "does-not-exist"))
		require.ErrorContains(t, err, "read migrations dir")
	})

	t.Run("broken migration file", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "broken-file.db")), &gorm.Config{})
		require.NoError(t, err)

		migrationsDir := t.TempDir()
		brokenLink := filepath.Join(migrationsDir, "001_missing.up.sql")
		require.NoError(t, os.Symlink(filepath.Join(migrationsDir, "missing.sql"), brokenLink))

		err = ApplyMigrations(context.Background(), db, migrationsDir)
		require.ErrorContains(t, err, "read migration 001_missing.up.sql")
	})

	t.Run("invalid sql", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "invalid-sql.db")), &gorm.Config{})
		require.NoError(t, err)

		migrationsDir := t.TempDir()
		require.NoError(t, writeFile(filepath.Join(migrationsDir, "001_invalid.up.sql"), []byte("THIS IS NOT SQL;")))

		err = ApplyMigrations(context.Background(), db, migrationsDir)
		require.ErrorContains(t, err, "apply migration 001_invalid.up.sql")
	})
}

func TestOpenDatabaseUsesInjectedDialectorAndErrors(t *testing.T) {
	originalDialector := mysqlDialector
	originalOpen := gormOpen
	originalSQLDBAccessor := sqlDBAccessor
	originalPingSQLDB := pingSQLDB
	t.Cleanup(func() {
		mysqlDialector = originalDialector
		gormOpen = originalOpen
		sqlDBAccessor = originalSQLDBAccessor
		pingSQLDB = originalPingSQLDB
	})

	var capturedDSN string
	mysqlDialector = func(dsn string) gorm.Dialector {
		capturedDSN = dsn
		return sqlite.Open(filepath.Join(t.TempDir(), "database.db"))
	}

	db, err := OpenDatabase(config.Config{MySQLDSN: "mysql://tenant-db"})
	require.NoError(t, err)
	require.Equal(t, "mysql://tenant-db", capturedDSN)
	require.NotNil(t, db)

	gormOpen = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	}

	_, err = OpenDatabase(config.Config{MySQLDSN: "mysql://tenant-db"})
	require.ErrorContains(t, err, "open mysql connection")
	require.ErrorContains(t, err, "open failed")

	gormOpen = originalOpen
	sqlDBAccessor = func(*gorm.DB) (*sql.DB, error) {
		return nil, errors.New("db handle failed")
	}

	_, err = OpenDatabase(config.Config{MySQLDSN: "mysql://tenant-db"})
	require.ErrorContains(t, err, "access sql db")
	require.ErrorContains(t, err, "db handle failed")

	sqlDBAccessor = originalSQLDBAccessor
	pingSQLDB = func(context.Context, *sql.DB) error {
		return errors.New("ping failed")
	}

	_, err = OpenDatabase(config.Config{MySQLDSN: "mysql://tenant-db"})
	require.ErrorContains(t, err, "ping mysql")
	require.ErrorContains(t, err, "ping failed")
}

func writeFile(path string, contents []byte) error {
	return os.WriteFile(path, contents, 0o644)
}
