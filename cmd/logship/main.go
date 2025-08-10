package main

import (
	"context"
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
)

func main() {
	cfg := config.FromEnv()

	// Ensure data dir exists
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "logship.db")
	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()
	log.Printf("store: using %s", dbPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := db.Init(ctx); err != nil {
		log.Fatalf("init store: %v", err)
	}

	// Start auto-trim to enforce max rows
	go db.StartAutoTrim(ctx, cfg.MaxRows, time.Minute)

	// Start syslog UDP/TCP if enabled
	if cfg.SyslogUDP != "" {
		go func() {
			if err := ingest.RunSyslogUDP(ctx, cfg.SyslogUDP, db, cfg); err != nil {
				log.Printf("syslog udp: %v", err)
			}
		}()
	}
	if cfg.SyslogTCP != "" {
		go func() {
			if err := ingest.RunSyslogTCP(ctx, cfg.SyslogTCP, db, cfg); err != nil {
				log.Printf("syslog tcp: %v", err)
			}
		}()
	}

	// HTTP API + UI
	mux := api.Router(db, cfg)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("http: listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	// Graceful shutdown
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	<-sigC
	log.Println("shutting down...")
	cancel()
	shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
}
