package store

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
)

type DB struct{ sql *sql.DB }

type Event struct {
	ID       int64     `json:"id"`
	TS       time.Time `json:"ts"`
	Host     string    `json:"host"`
	Level    string    `json:"level"`
	Message  string    `json:"message"`
	SourceIP string    `json:"source_ip"`
	Dropped  bool      `json:"dropped"`
}

type Stats struct {
	Total    int64 `json:"total"`
	LastHour int64 `json:"last_hour"`
	Dropped  int64 `json:"dropped"`
}

type QueryParams struct {
	Q       string
	Dropped *bool
	Limit   int
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?cache=shared&_pragma=journal_mode(WAL)&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	return &DB{sql: db}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) Init(ctx context.Context) error {
	_, err := d.sql.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS events(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TIMESTAMP NOT NULL,
  host TEXT,
  level TEXT,
  message TEXT,
  source_ip TEXT,
  dropped INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE INDEX IF NOT EXISTS idx_events_dropped_ts ON events(dropped, ts);
`)
	return err
}

func (d *DB) InsertEvent(ctx context.Context, e *Event) error {
	if e.TS.IsZero() {
		e.TS = time.Now().UTC()
	}
	res, err := d.sql.ExecContext(ctx, `
INSERT INTO events(ts,host,level,message,source_ip,dropped)
VALUES(?,?,?,?,?,?)
`, e.TS, e.Host, e.Level, e.Message, e.SourceIP, boolToInt(e.Dropped))
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	e.ID = id
	return nil
}

func (d *DB) InsertDrop(ctx context.Context, e *Event) error {
	e.Dropped = true
	return d.InsertEvent(ctx, e)
}

func (d *DB) QueryEvents(ctx context.Context, qp QueryParams) ([]Event, error) {
	args := []any{}
	where := "1=1"
	if qp.Q != "" {
		where += " AND (message LIKE ? OR host LIKE ? OR level LIKE ?)"
		arg := "%" + qp.Q + "%"
		args = append(args, arg, arg, arg)
	}
	if qp.Dropped != nil {
		where += " AND dropped = ?"
		args = append(args, boolToInt(*qp.Dropped))
	}
	limit := 100
	if qp.Limit > 0 && qp.Limit <= 1000 {
		limit = qp.Limit
	}
	rows, err := d.sql.QueryContext(ctx, `
SELECT id, ts, host, level, message, source_ip, dropped
FROM events
WHERE `+where+`
ORDER BY id DESC
LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var dropped int
		if err := rows.Scan(&e.ID, &e.TS, &e.Host, &e.Level, &e.Message, &e.SourceIP, &dropped); err != nil {
			return nil, err
		}
		e.Dropped = dropped == 1
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) Stats(ctx context.Context) (Stats, error) {
	var s Stats
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&s.Total); err != nil {
		return s, err
	}
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE dropped=1`).Scan(&s.Dropped); err != nil {
		return s, err
	}
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE ts >= datetime('now','-1 hour')`).Scan(&s.LastHour); err != nil {
		return s, err
	}
	return s, nil
}

func (d *DB) TrimToMaxRows(ctx context.Context, maxRows int) error {
	if maxRows <= 0 {
		return nil
	}
	// delete anything older than the newest maxRows
	_, err := d.sql.ExecContext(ctx, `
DELETE FROM events
WHERE id <= (
  SELECT COALESCE((SELECT id FROM events ORDER BY id DESC LIMIT 1 OFFSET ?), -1)
)
`, maxRows-1)
	return err
}

func (d *DB) StartAutoTrim(ctx context.Context, maxRows int, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = d.TrimToMaxRows(ctx, maxRows)
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
