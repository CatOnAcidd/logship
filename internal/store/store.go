package store

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Event struct {
	ID       int64           `json:"id"`
	TS       time.Time       `json:"ts"`
	Host     string          `json:"host"`
	Level    string          `json:"level"`
	Message  string          `json:"message"`
	SourceIP string          `json:"source_ip"`
	Dropped  bool            `json:"dropped"`
	RuleName string          `json:"rule_name,omitempty"`
	Raw      json.RawMessage `json:"raw,omitempty"`
}

type QueryParams struct {
	Dropped  *bool
	Level    string
	SourceIP string
	Search   string
	Limit    int
}

type DB struct {
	sql  *sql.DB
	path string
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "logship.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", dbPath)
	sdb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &DB{sql: sdb, path: dbPath}, nil
}

func (d *DB) Close() error { return d.sql.Close() }
func (d *DB) Sql() *sql.DB { return d.sql }

func (d *DB) Migrate(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := d.sql.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TIMESTAMP NOT NULL,
  host TEXT,
  level TEXT,
  message TEXT,
  source_ip TEXT,
  dropped INTEGER NOT NULL DEFAULT 0,
  rule_name TEXT,
  raw BLOB
);
CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
CREATE INDEX IF NOT EXISTS idx_events_dropped_ts ON events(dropped, ts);
CREATE TABLE IF NOT EXISTS rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  action TEXT NOT NULL,
  pattern TEXT NOT NULL
);
`)
	return err
}

func (d *DB) Insert(ctx context.Context, e *Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if e.TS.IsZero() {
		e.TS = time.Now().UTC()
	}
	res, err := d.sql.ExecContext(ctx, `INSERT INTO events(ts,host,level,message,source_ip,dropped,rule_name,raw)
	 VALUES(?,?,?,?,?,?,?,?)`, e.TS, e.Host, e.Level, e.Message, e.SourceIP, b2i(e.Dropped), e.RuleName, []byte(e.Raw))
	if err != nil {
		return err
	}
	e.ID, _ = res.LastInsertId()
	return nil
}

func (d *DB) InsertDrop(ctx context.Context, e *Event, rule string) error {
	e.Dropped = true
	e.RuleName = rule
	return d.Insert(ctx, e)
}

func (d *DB) Query(ctx context.Context, qp QueryParams) ([]Event, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	q := `SELECT id,ts,host,level,message,source_ip,dropped,rule_name,raw FROM events WHERE 1=1`
	var args []any
	if qp.Dropped != nil {
		q += ` AND dropped=?`
		args = append(args, b2i(*qp.Dropped))
	}
	if qp.Level != "" {
		q += ` AND level=?`
		args = append(args, qp.Level)
	}
	if qp.SourceIP != "" {
		q += ` AND source_ip=?`
		args = append(args, qp.SourceIP)
	}
	if qp.Search != "" {
		q += ` AND message LIKE ?`
		args = append(args, "%"+qp.Search+"%")
	}
	q += ` ORDER BY ts DESC`
	if qp.Limit <= 0 || qp.Limit > 1000 {
		qp.Limit = 100
	}
	q += ` LIMIT ?`
	args = append(args, qp.Limit)

	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		var raw []byte
		if err := rows.Scan(&e.ID, &e.TS, &e.Host, &e.Level, &e.Message, &e.SourceIP, &e.Dropped, &e.RuleName, &raw); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			e.Raw = json.RawMessage(raw)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) Stats(ctx context.Context) (ingested int64, dropped int64, byLevel map[string]int64, err error) {
	if ctx == nil { ctx = context.Background() }
	byLevel = map[string]int64{}

	row := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE dropped=0`)
	if err = row.Scan(&ingested); err != nil { return }

	row = d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE dropped=1`)
	if err = row.Scan(&dropped); err != nil { return }

	rows, err := d.sql.QueryContext(ctx, `SELECT COALESCE(level,''), COUNT(*) FROM events WHERE dropped=0 GROUP BY level`)
	if err != nil { return }
	defer rows.Close()
	for rows.Next() {
		var lvl string
		var c int64
		if err = rows.Scan(&lvl, &c); err != nil { return }
		byLevel[lvl] = c
	}
	err = rows.Err()
	return
}

type Rule struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Action  string `json:"action"`
	Pattern string `json:"pattern"`
}

func (d *DB) SaveRule(ctx context.Context, name, action, pattern string) error {
	if ctx == nil { ctx = context.Background() }
	if action != "drop" && action != "keep" {
		return errors.New("action must be drop or keep")
	}
	_, err := d.sql.ExecContext(ctx, `INSERT INTO rules(name,action,pattern) VALUES(?,?,?)`, name, action, pattern)
	return err
}

func (d *DB) ListRules(ctx context.Context) ([]Rule, error) {
	if ctx == nil { ctx = context.Background() }
	rows, err := d.sql.QueryContext(ctx, `SELECT id,name,action,pattern FROM rules ORDER BY id ASC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.Name, &r.Action, &r.Pattern); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func b2i(b bool) int { if b { return 1 } ; return 0 }
