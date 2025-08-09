package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/CatOnAcidd/logship/internal/config"
	"github.com/CatOnAcidd/logship/internal/store"
	"github.com/CatOnAcidd/logship/internal/transform"
)

type HTTPServer struct {
	cfg *config.Config
	srv *http.Server
}

func NewHTTPServer(cfg *config.Config, router *chi.Mux) *HTTPServer {
	return &HTTPServer{
		cfg: cfg,
		srv: &http.Server{
			Addr:    cfg.Server.HTTPListen,
			Handler: router,
		},
	}
}

func (h *HTTPServer) Start() error {
	log.Printf("http listening on %s", h.cfg.Server.HTTPListen)
	return h.srv.ListenAndServe()
}

func (h *HTTPServer) Stop(ctx context.Context) error { return h.srv.Shutdown(ctx) }

// API handlers

func IngestHandler(db *store.DB, tr *transform.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		dec.UseNumber()
		var payload any
		if err := dec.Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest); return
		}
		items := []map[string]any{}
		switch v := payload.(type) {
		case []any:
			for _, it := range v {
				m, ok := it.(map[string]any)
				if !ok { http.Error(w, "array must contain objects", 400); return }
				items = append(items, m)
			}
		case map[string]any:
			items = append(items, v)
		default:
			http.Error(w, "body must be object or array of objects", 400); return
		}
		for _, m := range items {
			raw, _ := json.Marshal(m)
			ev := &store.Event{
				Source: "http",
				Raw:    raw,
			}
			// Normalize timestamp if present
			if ts, ok := m["ts"]; ok {
				switch t := ts.(type) {
				case json.Number:
					if ms, err := t.Int64(); err==nil { ev.TS = ms }
				case float64:
					ev.TS = int64(t)
				}
			} else {
				ev.TS = time.Now().UnixMilli()
			}
			if host, ok := m["host"].(string); ok { ev.Host = host }
			if lvl, ok := m["level"].(string); ok { ev.Level = lvl }
			// Treat "event" as normalized payload if present
			if e, ok := m["event"]; ok {
				if b, err := json.Marshal(e); err==nil { ev.Event = b }
			}
			// Apply transform pipeline
			if tr != nil {
				out := tr.Apply(m)
				if out != nil {
					if b, err := json.Marshal(out); err==nil {
						ev.Transformed = b
					}
				}
			}
			if err := db.Insert(r.Context(), ev); err != nil {
				http.Error(w, fmt.Sprintf("insert error: %v", err), 500); return
			}
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"queued"}`))
	}
}

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	_, _ = w.Write([]byte("ok"))
}

func QueryLogsHandler(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var p store.QueryParams
		parseInt := func(name string) (int64, error) {
			if v := q.Get(name); v != "" {
				var n int64
				_, err := fmt.Sscan(v, &n)
				return n, err
			}
			return 0, errors.New("")
		}
		if v, err := parseInt("from"); err == nil { p.From = v }
		if v, err := parseInt("to"); err == nil { p.To = v }
		p.Q = q.Get("q")
		p.Level = q.Get("level")
		p.Source = q.Get("source")
		fmt.Sscan(q.Get("limit"), &p.Limit)

		items, err := db.Query(r.Context(), p)
		if err != nil {
			http.Error(w, err.Error(), 500); return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		_ = enc.Encode(items)
	}
}

// Router construction (shared with UI)
func Router(db *store.DB, cfg *config.Config, tr *transform.Engine) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/ingest", IngestHandler(db, tr))
	r.Get("/logs", QueryLogsHandler(db))
	r.Get("/healthz", HealthzHandler)
	return r
}
