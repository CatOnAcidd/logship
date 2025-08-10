package api

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/CatOnAcidd/logship/internal/store"
)

type Config struct {
	HTTPListen string
	DataDir    string

	SyslogUDP string
	SyslogTCP string

	MaxRows int

	Whitelist []string
	Blacklist []string
}

func LoadConfigFromEnv() Config {
	cfg := Config{
		HTTPListen: env("HTTP_LISTEN", ":8080"),
		DataDir: env("DATA_DIR", "/var/lib/logship"),
		SyslogUDP: os.Getenv("SYSLOG_UDP_LISTEN"),
		SyslogTCP: os.Getenv("SYSLOG_TCP_LISTEN"),
		MaxRows: envInt("MAX_ROWS", 500000),
	}
	if wl := strings.TrimSpace(os.Getenv("SOURCE_WHITELIST")); wl != "" {
		cfg.Whitelist = strings.Split(wl, ",")
	}
	if bl := strings.TrimSpace(os.Getenv("SOURCE_BLACKLIST")); bl != "" {
		cfg.Blacklist = strings.Split(bl, ",")
	}
	return cfg
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil { return i }
	}
	return def
}

func NewRouter(db *store.DB, cfg Config, webFS embed.FS) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP, middleware.RequestID, middleware.Logger, middleware.Recoverer)

	r.Mount("/api", apiRoutes(db, cfg))
	r.Handle("/*", http.FileServerFS(webFS))

	return r
}

func apiRoutes(db *store.DB, cfg Config) http.Handler {
	r := chi.NewRouter()

	r.Post("/ingest", func(w http.ResponseWriter, r *http.Request) {
		var ev store.Event
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			http.Error(w, err.Error(), 400); return
		}
		if ev.ReceivedAt.IsZero() { ev.ReceivedAt = store.Now() }
		if ev.Host == "" { ev.Host = r.RemoteAddr }
		if store.Blacklisted(cfg.Blacklist, ev.SourceIP) || !store.WhitelistedOrEmpty(cfg.Whitelist, ev.SourceIP) {
			_ = db.InsertDrop(r.Context(), ev)
			w.WriteHeader(202)
			return
		}
		if err := db.Insert(r.Context(), ev); err != nil { http.Error(w, err.Error(), 500); return }
		o := map[string]any{"status":"ok"}
		_ = json.NewEncoder(w).Encode(o)
	})

	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		q := store.QueryParams{
			Search: r.URL.Query().Get("q"),
			Limit:  envIntQ(r, "limit", 100),
			Offset: envIntQ(r, "offset", 0),
			OnlyDrops: r.URL.Query().Get("drops") == "1",
		}
		evs, total, err := db.Query(r.Context(), q)
		if err != nil { http.Error(w, err.Error(), 500); return }
		_ = json.NewEncoder(w).Encode(map[string]any{"total": total, "items": evs})
	})

	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		s, err := db.Stats(r.Context())
		if err != nil { http.Error(w, err.Error(), 500); return }
		_ = json.NewEncoder(w).Encode(s)
	})

	return r
}

func envIntQ(r *http.Request, k string, def int) int {
	if v := r.URL.Query().Get(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil { return i }
	}
	return def
}
