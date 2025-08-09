package ingest

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"time"

	"github.com/nxadm/tail"

	"github.com/CatOnAcidd/logship/internal/rules"
	"github.com/CatOnAcidd/logship/internal/store"
)

func RunFileTail(ctx context.Context, db *store.DB, path, glob string) {
	p := filepath.Join(path, glob)
	t, err := tail.TailFile(p, tail.Config{
		ReOpen: true, Follow: true, MustExist: false, Poll: true,
	})
	if err != nil { log.Printf("tail %s: %v", p, err); return }
	log.Printf("tailing %s", p)
	for {
		select {
		case <-ctx.Done():
			t.Cleanup(); t.Stop()
			return
		case line, ok := <-t.Lines:
			if !ok { time.Sleep(200 * time.Millisecond); continue }
			obj := map[string]any{"message": line.Text}
			raw, _ := json.Marshal(obj)
			ev := &store.Event{ Source: "file:"+p, Raw: raw, TS: time.Now().UnixMilli() }
			eng := rules.New(db.SQL()); _ = eng.Load(ctx)
			_ = evaluateAndStore(context.Background(), db, eng, ev)
		}
	}
}
