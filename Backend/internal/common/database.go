package common

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func databaseConfig() *gorm.Config {
	return &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			Colorful:                  true,
			IgnoreRecordNotFoundError: true,
			LogLevel:                  logger.Warn,
		}),
	}
}

// OpenDatabase keeps SQLite available for local tests while making the
// production PostgreSQL choice explicit and fail-fast.
func OpenDatabase(backend, path, dsn string) (*gorm.DB, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "sqlite":
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		return gorm.Open(sqlite.Open(path), databaseConfig())
	case "postgres", "postgresql":
		if strings.TrimSpace(dsn) == "" {
			return nil, fmt.Errorf("PostgreSQL backend requires a database DSN")
		}
		db, err := gorm.Open(postgres.Open(dsn), databaseConfig())
		if err != nil {
			return nil, err
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("configure PostgreSQL pool: %w", err)
		}
		configureSQLPool(sqlDB)
		return db, nil
	default:
		return nil, fmt.Errorf("unsupported database backend %q", backend)
	}
}

// configureSQLPool gives every Araneae process a bounded PostgreSQL pool.
// The shared cluster hosts control, executor, Orion, Vesper and CIS, so the
// default must be conservative; deployments can raise it deliberately with
// the same DATABASE_* settings when capacity is increased.
func configureSQLPool(db *sql.DB) {
	maxOpen := GetEnvInt("DATABASE_MAX_OPEN_CONNS", 4)
	if maxOpen < 1 {
		maxOpen = 4
	}
	maxIdle := GetEnvInt("DATABASE_MAX_IDLE_CONNS", 2)
	if maxIdle < 0 {
		maxIdle = 0
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	lifetimeSeconds := GetEnvInt("DATABASE_CONN_MAX_LIFETIME_SECONDS", 1800)
	if lifetimeSeconds < 0 {
		lifetimeSeconds = 0
	}
	idleSeconds := GetEnvInt("DATABASE_CONN_MAX_IDLE_SECONDS", 300)
	if idleSeconds < 0 {
		idleSeconds = 0
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(lifetimeSeconds) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(idleSeconds) * time.Second)
}
