package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/catonacidd/logship/internal/store"
)

type API struct {
	DB    *store.DB
	Opts  Opts
}

type Opts struct {
	RecentDefaultLimit int
}

func Register(r chi.Router, db *store.DB, opts Opts) {
	api := &API{DB: db, Opts: opts}
	r.Route("/api", func(r chi.Router) {
		r.Get("/stats", api.getStats)
		r.Get("/logs", api.getLogs)
	})
	r.Post("/ingest", api.httpIngest) // helpful for quick tests
}

func (a *API) getStats(w http.ResponseWriter, r *http.Request) {
	s, err := a.DB.Stats(r.Context())
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, s)
}

func (a *API) getLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dropped := q.Get("dropped") == "1"
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 { limit = a.Opts.RecentDefaultLimit }
	rows, err := a.DB.RecentLogs(r.Context(), dropped, limit)
	if err != nil { http.Error(w, err.Error(), 500); return }
	writeJSON(w, rows)
}

type ingestReq struct {
	Host    string `json:"host"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Raw     string `json:"raw"`
}

func (a *API) httpIngest(w http.ResponseWriter, r *http.Request) {
	var in ingestReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", 400); return
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" { ip = r.RemoteAddr }

	row := store.LogRow{
		TS:       time.Now().Unix(),
		Host:     in.Host,
		SourceIP: ip,
		Level:    in.Level,
		Message:  in.Message,
		Raw:      in.Raw,
		Dropped:  false,
	}
	if err := a.DB.InsertLog(context.Background(), row); err != nil {
		http.Error(w, err.Error(), 500); return
	}
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
