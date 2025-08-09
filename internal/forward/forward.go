package forward

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/CatOnAcidd/logship/internal/config"
	"github.com/CatOnAcidd/logship/internal/store"
	"github.com/CatOnAcidd/logship/internal/transform"
)

type Forwarder struct {
	db  *store.DB
	cfg *config.Config
	tr  *transform.Engine
	client *http.Client
}

func New(db *store.DB, cfg *config.Config, tr *transform.Engine) *Forwarder {
	return &Forwarder{
		db: db, cfg: cfg, tr: tr,
		client: &http.Client{ Timeout: 15 * time.Second },
	}
}

func (f *Forwarder) Run(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			f.tick(ctx)
		}
	}
}

func (f *Forwarder) tick(ctx context.Context) {
	if len(f.cfg.Forwarders) == 0 { return }
	dest := f.cfg.Forwarders[0] // MVP: single destination
	batchSize := 500
	if dest.Batch.Size > 0 { batchSize = dest.Batch.Size }
	events, err := f.db.FetchForForward(ctx, batchSize)
	if err != nil || len(events) == 0 { return }
	payload := make([]json.RawMessage, 0, len(events))
	ids := make([]int64, 0, len(events))
	for _, ev := range events {
		if len(ev.Transformed) > 0 {
			payload = append(payload, ev.Transformed)
		} else if len(ev.Event) > 0 {
			payload = append(payload, ev.Event)
		} else {
			payload = append(payload, ev.Raw)
		}
		ids = append(ids, ev.ID)
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", dest.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range dest.Header {
		req.Header.Set(k, v)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		log.Printf("forward error: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := f.db.MarkForwarded(ctx, ids); err != nil {
			log.Printf("mark forwarded: %v", err)
		}
	} else {
		log.Printf("forward bad status: %s", resp.Status)
	}
}
