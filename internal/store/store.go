package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

func ensureParent(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0o755)
}

// Normalize accepts either a directory or a filename; returns a full file path.
func normalizePath(p string) (string, error) {
	if p == "" {
		p = "/var/lib/logship"
	}
	// If it ends with a slash or is an existing dir, treat as directory.
	if strings.HasSuffix(p, string(os.PathSeparator)) {
		return filepath.Join(p, "logship.db"), nil
	}
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		return filepath.Join(p, "logship.db"), nil
	}
	return p, nil
}

func Open(path string) (*DB, error) {
	fp, err := normalizePath(path)
	if err != nil {
		return nil, fmt.Errorf("normalize db path: %w", err)
	}
	if err := ensureParent(fp); err != nil {
		return nil, fmt.Errorf("ensure parent dir: %w", err)
	}

	// Use a conservative DSN; WAL journaling can be enabled later if desired.
	dsn := fp + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &DB{sql: sqlDB}, nil
}
