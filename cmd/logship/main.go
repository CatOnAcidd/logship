package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/CatOnAcidd/logship/internal/api"
	"github.com/CatOnAcidd/logship/internal/ingest"
	"github.com/CatOnAcidd/logship/internal/store"
)

//go:embed web/*
var webFS embed.FS

func main() {
	cfg := api.LoadConfigFromEnv()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("mkdir data dir: %v", err)
	}

	db, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	go store.AutoTrimLoop(context.Background(), db, cfg.MaxRows)

	r := api.NewRouter(db, cfg, webFS)

	// start syslog listeners if configured
	if cfg.SyslogUDP != "" {
		go ingest.RunSyslogUDP(context.Background(), db, cfg.SyslogUDP)
	}
	if cfg.SyslogTCP != "" {
		go ingest.RunSyslogTCP(context.Background(), db, cfg.SyslogTCP)
	}

	srv := &http.Server{Addr: cfg.HTTPListen, Handler: r, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second}
	log.Printf("logship listening on %s (web), syslog udp=%q tcp=%q, data=%s", cfg.HTTPListen, cfg.SyslogUDP, cfg.SyslogTCP, cfg.DataDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server: %v", err)
	}
}
