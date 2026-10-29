// Package database opens the PostgreSQL connection and applies SQL migrations.
package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"financetracker/migrations"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a PostgreSQL connection pool and checks that it works.
func Connect(dsn string) (*gorm.DB, error) {
	// ParameterizedQueries keeps parameter values (e-mails, hashes) out of the logs.
	dbLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             500 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
	})
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		TranslateError: true, // turns PostgreSQL error codes into gorm.ErrDuplicatedKey etc.
		Logger:         dbLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database is not reachable: %w", err)
	}
	return db, nil
}

// Migrate applies every embedded *.sql file that has not been applied yet.
// Applied file names are remembered in the schema_migrations table.
func Migrate(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`).Error; err != nil {
		return err
	}
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		var count int64
		if err := db.Raw("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", e.Name()).Scan(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		content, err := migrations.FS.ReadFile(e.Name())
		if err != nil {
			return err
		}
		name := e.Name()
		err = db.Transaction(func(tx *gorm.DB) error {
			// Migration files must not contain semicolons except between statements.
			for _, stmt := range strings.Split(string(content), ";") {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				if err := tx.Exec(stmt).Error; err != nil {
					return err
				}
			}
			return tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", name).Error
		})
		if err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}
	}
	return nil
}
