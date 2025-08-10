// cmd/logship/main.go
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

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
)

func ensurePathIsDBFile(p string) (string, error) {
	if p == "" {
		p = "/var/lib/logship"
	}
	// If it's a dir (or doesn't exist yet), ensure dir and use /logship.db inside it.
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return "", err
		}
		return filepath.Join(p, "logship.db"), nil
	} else if os.IsNotExist(err) {
		// Treat as directory if it ends with a path separator or has no extension
		if filepath.Ext(p) == "" {
			if err := os.MkdirAll(p, 0o755); err != nil {
				return "", err
			}
			return filepath.Join(p, "logship.db"), nil
		}
		// Otherwise assume caller passed a filename; ensure parent exists
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		return p, nil
	}
	// Existing non-dir: assume it’s a file path
	if fi, _ := os.Stat(p); fi != nil && !fi.IsDir() {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		return p, nil
	}
	return filepath.Join(p, "logship.db"), nil
}

func main() {
	// Load config
	cfgPath := config.PathFromArgsOrDefault(os.Args[1:])
	cfg, _ := config.Load(cfgPath)

	// Resolve DB file path and ensure parent dir exists
	dbFile, err := ensurePathIsDBFile(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("resolve data path: %v", err)
	}

	// Open store (will create DB file if needed)
	db, err := store.Open(dbFile)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	_ = db // TODO: wire into your handlers/ingesters

	log.Printf("store: using %s", dbFile)

	// Minimal HTTP server to keep the process alive
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "logship is running")
	})

	addr := cfg.Server.HTTPListen
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
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
	// TODO: close DB, stop ingesters/forwarders cleanly when wired
}
