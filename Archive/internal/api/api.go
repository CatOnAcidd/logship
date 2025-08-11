package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/ingest"
	"github.com/catonacidd/logship/internal/store"
)

func Attach(mux *http.ServeMux, db *store.DB, cfg *config.Config) {
	// health
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// stats
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		s, err := db.Stats(r.Context())
		if err != nil {
			http.Error(w, "stats error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, s)
	})

	// events
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
				limit = n
			}
		}
		var dropped *bool
		if v := r.URL.Query().Get("dropped"); v == "true" || v == "1" {
			b := true
			dropped = &b
		}
		q := r.URL.Query().Get("q")
		evs, err := db.ListEvents(r.Context(), limit, dropped, q)
		if err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, evs)
	})

	// rules (Base: substring keep/drop)
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rs, err := db.ListRules(r.Context())
			if err != nil {
				http.Error(w, "rules error", http.StatusInternalServerError)
				return
			}
			writeJSON(w, rs)
		case http.MethodPost:
			var in store.Rule
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			if in.Action != "keep" && in.Action != "drop" {
				http.Error(w, "action must be keep|drop", http.StatusBadRequest)
				return
			}
			if in.Kind == "" {
				in.Kind = "substring"
			}
			if in.Kind != "substring" || in.Expr == "" {
				http.Error(w, "only substring rules with non-empty expr supported in base", http.StatusBadRequest)
				return
			}
			if err := db.AddRule(r.Context(), in); err != nil {
				http.Error(w, "save rule error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// HTTP ingest
	mux.Handle("/ingest", ingest.HandleHTTPIngest(db))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func WithLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	})
}
