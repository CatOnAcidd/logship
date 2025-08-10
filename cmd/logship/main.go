package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/catonacidd/logship/internal/api"
	"github.com/catonacidd/logship/internal/ingest"
	"github.com/catonacidd/logship/internal/store"
	"github.com/catonacidd/logship/internal/ui"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	// --- Config via envs (simple & explicit) ---
	listen := getenv("LISTEN_ADDR", ":8080")
	dataDir := getenv("DATA_DIR", "/var/lib/logship")
	dbPath := filepath.Join(dataDir, "logship.db")
	syslogUDP := os.Getenv("SYSLOG_UDP_LISTEN") // e.g. ":5514"
	syslogTCP := os.Getenv("SYSLOG_TCP_LISTEN") // e.g. ":5514"
	logLimitEnv := getenv("RECENT_LOG_LIMIT", "100")
	logLimit, _ := strconv.Atoi(logLimitEnv)
	if logLimit <= 0 || logLimit > 1000 {
		logLimit = 100
	}

	// --- Ensure data dir & open DB ---
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", dataDir, err)
	}
	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	log.Printf("store: using %s", dbPath)

	// --- Router: API + UI ---
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })
	api.Register(r, db, api.Opts{RecentDefaultLimit: logLimit})
	r.Mount("/", ui.Handler())

	srv := &http.Server{Addr: listen, Handler: r}

	// --- Syslog listeners (optional) ---
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if syslogUDP != "" || syslogTCP != "" {
		go ingest.RunSyslog(ctx, db, syslogUDP, syslogTCP)
	}

	// --- Start HTTP ---
	go func() {
		log.Printf("http: listening on %s", listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	// Graceful HTTP shutdown
	shCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shCtx)
	_ = db.Close()
}
