package ingest

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/CatOnAcidd/logship/internal/rules"
	"github.com/CatOnAcidd/logship/internal/store"
)

// evaluateAndStore applies rules (drop/allow) and inserts event; stores drop preview when needed.
func evaluateAndStore(ctx context.Context, db *store.DB, eng *rules.Engine, ev *store.Event) error {
	// Build probe (simple for now: message := string(raw))
	probe := rules.EventProbe{
		TS: ev.TS, Source: ev.Source, Host: ev.Host, Level: ev.Level,
		Message: string(ev.Raw),
	}
	dec := eng.Evaluate(probe)
	if dec.Action == rules.ActionDrop {
		ev.Dropped = true
		// Add to drops buffer
		js := map[string]any{"host":ev.Host,"level":ev.Level,"message":probe.Message}
		b, _ := json.Marshal(js)
		if err := db.InsertDrop(ctx, time.Now().UnixMilli(), ev.Source, dec.RuleName, string(b)); err!=nil {
			log.Printf("insert drop preview: %v", err)
		}
	}
	return db.Insert(ctx, ev)
}
