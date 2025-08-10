package store

import "database/sql"

func migrate(db *sql.DB) error {
	// Pragmas per connection
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		return err
	}

	// Tables & indexes
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS logs (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			ts        INTEGER NOT NULL,
			host      TEXT,
			source_ip TEXT,
			level     TEXT,
			message   TEXT,
			raw       TEXT,
			dropped   INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_logs_ts ON logs(ts);`,
		`CREATE INDEX IF NOT EXISTS idx_logs_drop_ts ON logs(dropped, ts);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
