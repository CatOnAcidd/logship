package store

import (
	"context"
)

type LogRow struct {
	ID       int64  `json:"id"`
	TS       int64  `json:"ts"`
	Host     string `json:"host"`
	SourceIP string `json:"source_ip"`
	Level    string `json:"level"`
	Message  string `json:"message"`
	Raw      string `json:"raw"`
	Dropped  bool   `json:"dropped"`
}

func boolToInt(b bool) int { if b { return 1 } ; return 0 }

func (d *DB) InsertLog(ctx context.Context, lr LogRow) error {
	_, err := d.ExecContext(ctx, `
		INSERT INTO logs(ts,host,source_ip,level,message,raw,dropped)
		VALUES(?,?,?,?,?,?,?)`,
		lr.TS, lr.Host, lr.SourceIP, lr.Level, lr.Message, lr.Raw, boolToInt(lr.Dropped))
	return err
}

func (d *DB) RecentLogs(ctx context.Context, dropped bool, limit int) ([]LogRow, error) {
	if limit <= 0 || limit > 1000 { limit = 100 }
	rows, err := d.QueryContext(ctx, `
		SELECT id, ts, host, source_ip, level, message, raw, dropped
		FROM logs
		WHERE dropped=?
		ORDER BY ts DESC
		LIMIT ?`, boolToInt(dropped), limit)
	if err != nil { return nil, err }
	defer rows.Close()

	out := make([]LogRow, 0, limit)
	for rows.Next() {
		var lr LogRow
		var di int
		if err := rows.Scan(&lr.ID, &lr.TS, &lr.Host, &lr.SourceIP, &lr.Level, &lr.Message, &lr.Raw, &di); err != nil {
			return nil, err
		}
		lr.Dropped = di == 1
		out = append(out, lr)
	}
	return out, rows.Err()
}

type Stats struct {
	TotalIngested int64 `json:"total_ingested"`
	TotalDropped  int64 `json:"total_dropped"`
}

func (d *DB) Stats(ctx context.Context) (Stats, error) {
	var s Stats
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM logs WHERE dropped=0`).Scan(&s.TotalIngested); err != nil {
		return s, err
	}
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM logs WHERE dropped=1`).Scan(&s.TotalDropped); err != nil {
		return s, err
	}
	return s, nil
}
