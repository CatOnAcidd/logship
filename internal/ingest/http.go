package ingest

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
)

type IngestRequest struct {
	Host     string `json:"host"`
	Level    string `json:"level"`
	Message  string `json:"message"`
	SourceIP string `json:"source_ip"`
}

func HandleHTTPIngest(db *store.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req IngestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		req.Host = strings.TrimSpace(req.Host)
		req.Level = strings.TrimSpace(req.Level)
		if req.Level == "" {
			req.Level = "info"
		}
		e := store.Event{
			Host:     req.Host,
			Level:    req.Level,
			Message:  req.Message,
			SourceIP: req.SourceIP,
		}
		if e.SourceIP == "" {
			e.SourceIP = r.Header.Get("X-Forwarded-For")
			if e.SourceIP == "" {
				e.SourceIP = strings.Split(r.RemoteAddr, ":")[0]
			}
		}
		if err := evaluateAndStore(r.Context(), db, cfg, e); err != nil {
			http.Error(w, "store error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}

func HandleQuery(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconvAtoi(v); err == nil && n > 0 && n <= 1000 {
				limit = n
			}
		}
		var dropped *bool
		if v := r.URL.Query().Get("dropped"); v != "" {
			if v == "1" || strings.ToLower(v) == "true" {
				t := true
				dropped = &t
			} else {
				f := false
				dropped = &f
			}
		}
		evs, err := db.QueryEvents(r.Context(), store.QueryParams{Q: q, Limit: limit, Dropped: dropped})
		if err != nil {
			http.Error(w, "query error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, evs)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func strconvAtoi(s string) (int, error) {
	var n int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, &strconvErr{}
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

type strconvErr struct{}

func (e *strconvErr) Error() string { return "atoi: invalid" }
