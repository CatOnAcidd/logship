package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catonacidd/logship/internal/api"
	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/ingest"
	"github.com/catonacidd/logship/internal/store"
)

func main() {
	cfg := config.Load(config.PathFromArgsOrDefault(os.Args[1:]))

	db, err := store.Open(cfg.Storage.DataDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(nil); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	re, err := ingest.NewRulesEngine(db)
	if err != nil {
		log.Fatalf("rules: %v", err)
	}

	sys := ingest.SyslogServer{
		DB:        db,
		Engine:    re,
		IPFilter:  ingest.NewIPFilter(cfg.Lists),
		UDPListen: cfg.Server.SyslogUDP,
		TCPListen: cfg.Server.SyslogTCP,
	}
	sys.Start()

	router := api.NewRouter(db, re, cfg)
	srv := &http.Server{
		Addr:         cfg.Server.ListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("http: listening on %s", cfg.Server.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	_ = srv.Close()
	sys.Shutdown()
}
