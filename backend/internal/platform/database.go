package platform

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/wecrazy/sociomile/backend/internal/config"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	mysqlDialector = func(dsn string) gorm.Dialector {
		return gormmysql.Open(dsn)
	}
	gormOpen = func(dialector gorm.Dialector, options ...gorm.Option) (*gorm.DB, error) {
		return gorm.Open(dialector, options...)
	}
	sqlDBAccessor = func(db *gorm.DB) (*sql.DB, error) {
		return db.DB()
	}
	pingSQLDB = func(ctx context.Context, db *sql.DB) error {
		return db.PingContext(ctx)
	}
)

// OpenDatabase opens the primary MySQL connection used by backend services.
func OpenDatabase(cfg config.Config) (*gorm.DB, error) {
	db, err := gormOpen(mysqlDialector(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	sqlDB, err := sqlDBAccessor(db)
	if err != nil {
		return nil, fmt.Errorf("access sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pingSQLDB(ctx, sqlDB); err != nil {
		return nil, classifyMySQLError(err)
	}

	return db, nil
}

func classifyMySQLError(err error) error {
	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1045:
			return fmt.Errorf("ping mysql: access denied; verify MYSQL_USER and MYSQL_PASSWORD, or recreate the local MySQL volume so the container can provision the user from env again: %w", err)
		case 1049:
			return fmt.Errorf("ping mysql: unknown database; verify MYSQL_DATABASE, or recreate the local MySQL volume so the container can provision the database from env again: %w", err)
		}
	}

	return fmt.Errorf("ping mysql: %w", err)
}

// ApplyMigrations runs each pending SQL migration in filename order.
func ApplyMigrations(ctx context.Context, db *gorm.DB, dir string) error {
	if err := db.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		files = append(files, entry.Name())
	}

	sort.Strings(files)

	for _, name := range files {
		var applied int64
		if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM schema_migrations WHERE name = ?", name).Scan(&applied).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}

		if applied > 0 {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		statement := strings.TrimSpace(string(contents))
		if statement == "" {
			continue
		}

		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}

			return tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", name).Error
		}); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}
