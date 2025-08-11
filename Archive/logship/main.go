package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/catonacidd/logship/internal/api"
	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/ingest"
	"github.com/catonacidd/logship/internal/store"
	"github.com/catonacidd/logship/internal/ui"
)

func main() {
	cfg := config.FromEnv()

	// Ensure data dir
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}

	// Open DB
	dbPath := filepath.Join(cfg.DataDir, "logship.db")
	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()
	log.Printf("store: using %s", dbPath)

	// Rule engine lives in DB (base: simple substring rules)
	// Start syslog listeners
	syslog := ingest.NewSyslogIngest(db, cfg)
	if err := syslog.Start(); err != nil {
		log.Fatalf("syslog start: %v", err)
	}
	defer syslog.Close()

	// File tail (optional; disabled by default)
	if cfg.FileTailPath != "" {
		go ingest.RunFileTail(db, cfg.FileTailPath, cfg.FileTailGlob)
	}

	// HTTP API + UI
	mux := http.NewServeMux()
	api.Attach(mux, db, cfg)

	uih, _ := ui.Handler()
	mux.Handle("/", uih)

	srv := &http.Server{
		Addr:              cfg.HTTPListen,
		Handler:           api.WithLogging(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("http: listening on %s", cfg.HTTPListen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	// Wait for signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Printf("shutting down…")
	_ = srv.Close()
}
