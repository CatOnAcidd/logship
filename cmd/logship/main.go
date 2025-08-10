package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
	"github.com/catonacidd/logship/internal/ui"
)

func ensureDB(p string) (string, error) {
	if p == "" {
		p = "/var/lib/logship"
	}
	// if looks like a dir (or is missing), make dir and use /logship.db
	fi, err := os.Stat(p)
	switch {
	case err == nil && fi.IsDir():
		if err := os.MkdirAll(p, 0o755); err != nil { return "", err }
		return filepath.Join(p, "logship.db"), nil
	case os.IsNotExist(err):
		// treat as dir if no extension
		if filepath.Ext(p) == "" {
			if err := os.MkdirAll(p, 0o755); err != nil { return "", err }
			return filepath.Join(p, "logship.db"), nil
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { return "", err }
		return p, nil
	default:
		// existing non-dir => assume file
		if !fi.IsDir() { _ = os.MkdirAll(filepath.Dir(p), 0o755); return p, nil }
		return filepath.Join(p, "logship.db"), nil
	}
}

func main() {
	// Load config
	cfgPath := config.PathFromArgsOrDefault(os.Args[1:])
	cfg, _ := config.Load(cfgPath)

	// Open store
	dbFile, err := ensureDB(cfg.Storage.Path)
	if err != nil { log.Fatalf("resolve data path: %v", err) }
	db, err := store.Open(dbFile)
	if err != nil { log.Fatalf("open store: %v", err) }
	defer func() { _ = db }()

	log.Printf("store: using %s", dbFile)

	// Router
	r := chi.NewRouter()

	// Health
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// TODO: mount your API under /api here

	// UI (embedded)
	r.Mount("/", ui.Handler())

	addr := cfg.Server.HTTPListen
	if addr == "" { addr = ":8080" }

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("http: listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown: signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http: shutdown error: %v", err)
	}
	fmt.Println("bye")
}
