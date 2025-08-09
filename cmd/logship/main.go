package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CatOnAcidd/logship/internal/config"
	"github.com/CatOnAcidd/logship/internal/forward"
	"github.com/CatOnAcidd/logship/internal/ingest"
	"github.com/CatOnAcidd/logship/internal/store"
	"github.com/CatOnAcidd/logship/internal/transform"
	"github.com/CatOnAcidd/logship/internal/ui"
)

func main() {
	cfgPath := config.PathFromArgsOrDefault()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := store.Open(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer db.Close()

	tr, err := transform.New(cfg)
	if err != nil {
		log.Fatalf("transform init: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Forwarder
	fw := forward.New(db, cfg, tr)
	go fw.Run(ctx)

	// Ingest HTTP + UI
	router := ui.NewRouter(db, cfg, tr)
	httpSrv := ingest.NewHTTPServer(cfg, router)
	go func() {
		if err := httpSrv.Start(); err != nil {
			log.Fatalf("http server: %v", err)
		}
	}()

	// Syslog TCP/UDP
	if cfg.Server.SyslogTCPListen != "" || cfg.Server.SyslogUDPListen != "" {
		go ingest.RunSyslog(ctx, db, cfg)
	}

	// File tails
	for _, ft := range cfg.Server.FileTails {
		go ingest.RunFileTail(ctx, db, ft.Path, ft.Glob)
	}

	// Handle signals
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc

	log.Println("shutting down...")
	cancel()
	time.Sleep(500 * time.Millisecond)
	httpSrv.Stop(context.Background())
}
