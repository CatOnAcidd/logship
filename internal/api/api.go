package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/ingest"
	"github.com/catonacidd/logship/internal/store"
	"github.com/catonacidd/logship/internal/ui"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(db *store.DB, re *ingest.RulesEngine, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP, middleware.RequestID, middleware.Recoverer)

	h := &ingest.HTTPIngest{DB: db, Engine: re, IPFilter: ingest.NewIPFilter(cfg.Lists)}
	r.Post("/ingest", h.HandleIngest)

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{"ok": true})
		})
		api.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			ing, drop, byLvl, err := db.Stats(r.Context())
			if err != nil { http.Error(w, err.Error(), 500); return }
			writeJSON(w, map[string]any{"ingested": ing, "dropped": drop, "levels": byLvl})
		})
		api.Get("/events", func(w http.ResponseWriter, r *http.Request) {
			q := store.QueryParams{
				Limit:    getInt(r, "limit", cfg.Storage.RecentN),
				Level:    r.URL.Query().Get("level"),
				SourceIP: r.URL.Query().Get("source_ip"),
				Search:   r.URL.Query().Get("search"),
			}
			if v := r.URL.Query().Get("dropped"); v != "" {
				b := v == "true" || v == "1"
				q.Dropped = &b
			}
			evs, err := db.Query(r.Context(), q)
			if err != nil { http.Error(w, err.Error(), 500); return }
			writeJSON(w, evs)
		})
		api.Post("/rules", func(w http.ResponseWriter, r *http.Request) {
			var in struct{ Name, Action, Pattern string }
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil { http.Error(w, "bad json", 400); return }
			if err := db.SaveRule(r.Context(), in.Name, in.Action, in.Pattern); err != nil { http.Error(w, err.Error(), 400); return }
			_ = re.Add(in.Name, in.Action, in.Pattern)
			writeJSON(w, map[string]any{"ok": true})
		})
		api.Get("/rules", func(w http.ResponseWriter, r *http.Request) {
			rl, err := db.ListRules(r.Context())
			if err != nil { http.Error(w, err.Error(), 500); return }
			writeJSON(w, rl)
		})
		api.Post("/rules/test", func(w http.ResponseWriter, r *http.Request) {
			var in struct{ Message string }
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil { http.Error(w, "bad json", 400); return }
			ev := &store.Event{Message: in.Message}
			drop, rule := re.Evaluate(ev)
			writeJSON(w, map[string]any{"drop": drop, "rule": rule})
		})
	})

	ui.Mount(r)
	go pruneLoop(db, cfg.Storage.MaxMB)
	return r
}

func pruneLoop(db *store.DB, maxMB int) {
	if maxMB <= 0 { return }
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		_ = pruneOnce(db, maxMB)
	}
}

func pruneOnce(db *store.DB, maxMB int) error {
	ctx := context.Background()
	var pageCount, pageSize int64
	row := db.Sql().QueryRowContext(ctx, "PRAGMA page_count")
	if err := row.Scan(&pageCount); err != nil { return nil }
	row = db.Sql().QueryRowContext(ctx, "PRAGMA page_size")
	if err := row.Scan(&pageSize); err != nil { return nil }
	if pageCount*pageSize <= int64(maxMB)*1024*1024 {
		return nil
	}
	_, _ = db.Sql().ExecContext(ctx, `DELETE FROM events WHERE id IN (
		SELECT id FROM events ORDER BY ts ASC LIMIT (SELECT COUNT(*)/20 FROM events)
	)`)
	return nil
}

func getInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" { return def }
	if i, err := strconv.Atoi(v); err == nil { return i }
	return def
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
