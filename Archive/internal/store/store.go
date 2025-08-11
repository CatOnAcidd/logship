package store

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
)

type DB struct {
	sql *sql.DB
}

type Event struct {
	ID      int64     `json:"id"`
	TS      time.Time `json:"ts"`
	Host    string    `json:"host"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
	Dropped bool      `json:"dropped"`
}

type Rule struct {
	ID     int64  `json:"id"`
	Action string `json:"action"` // "keep" or "drop"
	Kind   string `json:"kind"`   // "substring" (base)
	Expr   string `json:"expr"`
}

type Stats struct {
	Received  int64 `json:"received"`
	Forwarded int64 `json:"forwarded"`
	Dropped   int64 `json:"dropped"`
}

func Open(path string) (*DB, error) {
	d, err := sql.Open("sqlite", path+"?_busy_timeout=5000&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if err := migrate(d); err != nil {
		d.Close()
		return nil, err
	}
	return &DB{sql: d}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS events(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ts INTEGER NOT NULL,
			host TEXT,
			level TEXT,
			message TEXT,
			dropped INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);`,
		`CREATE INDEX IF NOT EXISTS idx_events_dropped ON events(dropped);`,
		`CREATE TABLE IF NOT EXISTS rules(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			kind TEXT NOT NULL,
			expr TEXT NOT NULL
		);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) InsertEvent(ctx context.Context, e *Event) error {
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO events(ts,host,level,message,dropped) VALUES(?,?,?,?,?)`,
		e.TS.Unix(), e.Host, e.Level, e.Message, boolToInt(e.Dropped))
	return err
}

func (d *DB) ListEvents(ctx context.Context, limit int, dropped *bool, q string) ([]Event, error) {
	args := []any{}
	where := "WHERE 1=1"
	if dropped != nil {
		where += " AND dropped=?"
		args = append(args, boolToInt(*dropped))
	}
	if q != "" {
		where += " AND (host LIKE ? OR message LIKE ?)"
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	args = append(args, limit)
	rows, err := d.sql.QueryContext(ctx, `SELECT id, ts, host, level, message, dropped
		FROM events `+where+` ORDER BY id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var unix int64
		var droppedInt int
		if err := rows.Scan(&e.ID, &unix, &e.Host, &e.Level, &e.Message, &droppedInt); err != nil {
			return nil, err
		}
		e.TS = time.Unix(unix, 0).UTC()
		e.Dropped = droppedInt == 1
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) Stats(ctx context.Context) (Stats, error) {
	var s Stats
	// Received = all rows, Dropped = dropped=1
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&s.Received); err != nil {
		return s, err
	}
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE dropped=1`).Scan(&s.Dropped); err != nil {
		return s, err
	}
	// Base edition: Forwarded not implemented => 0
	return s, nil
}

func (d *DB) AddRule(ctx context.Context, r Rule) error {
	_, err := d.sql.ExecContext(ctx, `INSERT INTO rules(action,kind,expr) VALUES(?,?,?)`, r.Action, r.Kind, r.Expr)
	return err
}

func (d *DB) ListRules(ctx context.Context) ([]Rule, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id, action, kind, expr FROM rules ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.Action, &r.Kind, &r.Expr); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
