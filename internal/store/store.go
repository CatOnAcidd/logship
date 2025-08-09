package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

type Event struct {
	ID          int64           `json:"id"`
	TS          int64           `json:"ts"`
	Source      string          `json:"source"`
	Level       string          `json:"level,omitempty"`
	Host        string          `json:"host,omitempty"`
	Raw         json.RawMessage `json:"raw"`
	Event       json.RawMessage `json:"event,omitempty"`
	Transformed json.RawMessage `json:"transformed,omitempty"`
	Forwarded   bool            `json:"forwarded"`
	RetryCount  int             `json:"retry_count"`
}

func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared&_pragma=busy_timeout=5000&_pragma=journal_mode(WAL)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db := &DB{sql: sqlDB}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) migrate() error {
	_, err := d.sql.Exec(`
CREATE TABLE IF NOT EXISTS logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts INTEGER NOT NULL,
  source TEXT NOT NULL,
  level TEXT,
  host TEXT,
  raw TEXT NOT NULL,
  event TEXT,
  transformed TEXT,
  forwarded INTEGER NOT NULL DEFAULT 0,
  retry_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS ix_logs_ts ON logs(ts);
CREATE INDEX IF NOT EXISTS ix_logs_forwarded ON logs(forwarded, ts);
`)
	return err
}

func (d *DB) Insert(ctx context.Context, ev *Event) error {
	if ev.TS == 0 {
		ev.TS = time.Now().UnixMilli()
	}
	_, err := d.sql.ExecContext(ctx, `INSERT INTO logs (ts,source,level,host,raw,event,transformed,forwarded,retry_count)
VALUES (?,?,?,?,?,?,?,?,?)`, ev.TS, ev.Source, ev.Level, ev.Host, string(ev.Raw), bytesOrNil(ev.Event), bytesOrNil(ev.Transformed), boolToInt(ev.Forwarded), ev.RetryCount)
	return err
}

func bytesOrNil(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

func boolToInt(b bool) int { if b { return 1 }; return 0 }

func (d *DB) FetchForForward(ctx context.Context, limit int) ([]Event, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT id,ts,source,level,host,COALESCE(transformed,event,raw) FROM logs WHERE forwarded=0 ORDER BY ts ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var ev Event
		var payload string
		if err := rows.Scan(&ev.ID, &ev.TS, &ev.Source, &ev.Level, &ev.Host, &payload); err != nil {
			return nil, err
		}
		ev.Transformed = json.RawMessage(payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (d *DB) MarkForwarded(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE logs SET forwarded=1 WHERE id=?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type QueryParams struct {
	From  int64
	To    int64
	Q     string
	Level string
	Source string
	Limit int
}

func (d *DB) Query(ctx context.Context, p QueryParams) ([]Event, error) {
	if p.Limit <= 0 || p.Limit > 1000 {
		p.Limit = 200
	}
	q := `SELECT id,ts,source,level,host,raw,event,transformed,forwarded,retry_count FROM logs WHERE 1=1`
	args := []any{}
	if p.From > 0 {
		q += ` AND ts >= ?`; args = append(args, p.From)
	}
	if p.To > 0 {
		q += ` AND ts <= ?`; args = append(args, p.To)
	}
	if p.Level != "" {
		q += ` AND level = ?`; args = append(args, p.Level)
	}
	if p.Source != "" {
		q += ` AND source LIKE ?`; args = append(args, p.Source+"%")
	}
	if p.Q != "" {
		// naive LIKE across raw/event/transformed
		q += ` AND (raw LIKE ? OR event LIKE ? OR transformed LIKE ?)`
		for i:=0;i<3;i++ { args = append(args, "%"+p.Q+"%") }
	}
	q += ` ORDER BY ts DESC LIMIT ?`; args = append(args, p.Limit)
	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var ev Event
		var raw, event, transformed *string
		if err := rows.Scan(&ev.ID,&ev.TS,&ev.Source,&ev.Level,&ev.Host,&raw,&event,&transformed,&ev.Forwarded,&ev.RetryCount); err != nil {
			return nil, err
		}
		if raw!=nil { ev.Raw = json.RawMessage(*raw) }
		if event!=nil { ev.Event = json.RawMessage(*event) }
		if transformed!=nil { ev.Transformed = json.RawMessage(*transformed) }
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (d *DB) UpdateTransformed(ctx context.Context, id int64, payload []byte) error {
	res, err := d.sql.ExecContext(ctx, `UPDATE logs SET transformed=? WHERE id=?`, string(payload), id)
	if err != nil { return err }
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return errors.New("no rows updated")
	}
	return nil
}
