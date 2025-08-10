package ingest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/catonacidd/logship/internal/store"
)

type HTTPIngest struct {
	DB       *store.DB
	Engine   *RulesEngine
	IPFilter *IPFilter
}

type payload struct {
	Host    string          `json:"host"`
	Level   string          `json:"level"`
	Message string          `json:"message"`
	Raw     json.RawMessage `json:"raw,omitempty"`
}

func (h *HTTPIngest) HandleIngest(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		http.Error(w, "invalid json", 400); return
	}
	srcIP := clientIP(r.RemoteAddr)
	ev := &store.Event{
		TS:       time.Now().UTC(),
		Host:     p.Host,
		Level:    p.Level,
		Message:  p.Message,
		SourceIP: srcIP,
		Raw:      p.Raw,
	}
	if h.IPFilter != nil && !h.IPFilter.Allowed(srcIP) {
		_ = h.DB.InsertDrop(r.Context(), ev, "ip-blacklist")
		w.WriteHeader(http.StatusAccepted); _, _ = w.Write([]byte(`{"ok":true}`)); return
	}
	drop, rule := h.Engine.Evaluate(ev)
	if drop { _ = h.DB.InsertDrop(context.Background(), ev, rule) } else { _ = h.DB.Insert(context.Background(), ev) }
	w.WriteHeader(http.StatusAccepted); _, _ = w.Write([]byte(`{"ok":true}`))
}

func clientIP(remote string) string {
	if i := strings.LastIndex(remote, ":"); i > 0 { return remote[:i] }
	return remote
}
