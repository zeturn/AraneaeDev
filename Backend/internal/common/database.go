package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// OpenDatabase keeps SQLite available for local tests while making the
// production PostgreSQL choice explicit and fail-fast.
func OpenDatabase(backend, path, dsn string) (*gorm.DB, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "sqlite":
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		return gorm.Open(sqlite.Open(path), &gorm.Config{})
	case "postgres", "postgresql":
		if strings.TrimSpace(dsn) == "" {
			return nil, fmt.Errorf("PostgreSQL backend requires a database DSN")
		}
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database backend %q", backend)
	}
}
