package store

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"path/filepath"
	"time"
)

type DB struct {
	conn *sql.DB
}

func Open(dataDir string) (*DB, error) {
	p := filepath.Join(dataDir, "logship.db")
	c, err := sql.Open("sqlite", p+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil { return nil, err }
	if _, err := c.Exec(schemaDDL); err != nil { return nil, err }
	return &DB{conn: c}, nil
}

func (d *DB) Close() error { return d.conn.Close() }

type Event struct {
	ID int64 `json:"id"`
	Host string `json:"host"`
	Level string `json:"level"`
	Message string `json:"message"`
	SourceIP string `json:"source_ip"`
	ReceivedAt time.Time `json:"received_at"`
}

type QueryParams struct {
	Search string
	Limit int
	Offset int
	OnlyDrops bool
}

func Now() time.Time { return time.Now().UTC() }

const schemaDDL = `
CREATE TABLE IF NOT EXISTS events(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	host TEXT,
	level TEXT,
	message TEXT,
	source_ip TEXT,
	received_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_events_received_at ON events(received_at);
CREATE TABLE IF NOT EXISTS drops(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	host TEXT, level TEXT, message TEXT, source_ip TEXT, received_at TIMESTAMP
);
`

func (d *DB) Insert(ctx context.Context, ev Event) error {
	_, err := d.conn.ExecContext(ctx, `INSERT INTO events(host,level,message,source_ip,received_at)VALUES(?,?,?,?,?)`,
		ev.Host, ev.Level, ev.Message, ev.SourceIP, ev.ReceivedAt)
	return err
}

func (d *DB) InsertDrop(ctx context.Context, ev Event) error {
	_, err := d.conn.ExecContext(ctx, `INSERT INTO drops(host,level,message,source_ip,received_at)VALUES(?,?,?,?,?)`,
		ev.Host, ev.Level, ev.Message, ev.SourceIP, ev.ReceivedAt)
	return err
}

func (d *DB) Query(ctx context.Context, q QueryParams) ([]Event, int, error) {
	table := "events"
	if q.OnlyDrops { table = "drops" }
	countSQL := "SELECT COUNT(*) FROM " + table
	dataSQL := "SELECT id,host,level,message,source_ip,received_at FROM " + table + " ORDER BY id DESC LIMIT ? OFFSET ?"
	args := []any{q.Limit, q.Offset}
	if q.Search != "" {
		like := "%" + q.Search + "%"
		countSQL = "SELECT COUNT(*) FROM " + table + " WHERE host LIKE ? OR level LIKE ? OR message LIKE ? OR source_ip LIKE ?"
		dataSQL = "SELECT id,host,level,message,source_ip,received_at FROM " + table + " WHERE host LIKE ? OR level LIKE ? OR message LIKE ? OR source_ip LIKE ? ORDER BY id DESC LIMIT ? OFFSET ?"
		args = []any{like, like, like, like, q.Limit, q.Offset}
	}
	var total int
	if err := d.conn.QueryRowContext(ctx, countSQL, args[:len(args)-2]...).Scan(&total); err != nil { return nil, 0, err }
	rows, err := d.conn.QueryContext(ctx, dataSQL, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var ev Event
		if err := rows.Scan(&ev.ID, &ev.Host, &ev.Level, &ev.Message, &ev.SourceIP, &ev.ReceivedAt); err != nil { return nil, 0, err }
		out = append(out, ev)
	}
	return out, total, nil
}

type StatsResp struct {
	Total int `json:"total"`
	Drops int `json:"drops"`
}

func (d *DB) Stats(ctx context.Context) (StatsResp, error) {
	var s StatsResp
	if err := d.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&s.Total); err != nil { return s, err }
	if err := d.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM drops`).Scan(&s.Drops); err != nil { return s, err }
	return s, nil
}

// AutoTrimLoop keeps only the most recent N rows across tables to bound disk usage.
func AutoTrimLoop(ctx context.Context, d *DB, maxRows int) {
	if maxRows <= 0 { maxRows = 500000 }
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, table := range []string{"events", "drops"} {
				// delete rows older than the N newest
				_, _ = d.conn.ExecContext(ctx, `DELETE FROM `+table+` WHERE id < IFNULL((SELECT id FROM `+table+` ORDER BY id DESC LIMIT 1 OFFSET ?), 0)`, maxRows)
			}
		}
	}
}

// helpers for whitelist/blacklist
func WhitelistedOrEmpty(list []string, ip string) bool {
	if len(list) == 0 { return true }
	for _, s := range list { if s == ip { return true } }
	return false
}
func Blacklisted(list []string, ip string) bool {
	for _, s := range list { if s == ip { return true } }
	return false
}
