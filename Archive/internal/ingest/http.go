package ingest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/catonacidd/logship/internal/store"
)

type IngestRequest struct {
	Host    string `json:"host"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

func HandleHTTPIngest(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in IngestRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		e := &store.Event{
			TS:      time.Now().UTC(),
			Host:    in.Host,
			Level:   in.Level,
			Message: in.Message,
		}
		rules, _ := db.ListRules(r.Context())
		if err := evaluateAndInsert(r.Context(), db, e, rules); err != nil {
			http.Error(w, "store error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
