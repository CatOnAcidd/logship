package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/catonacidd/logship/internal/config"
	"github.com/catonacidd/logship/internal/store"
)

func main() {
	cfgPath := config.PathFromArgsOrDefault(os.Args[1:])
	cfg, _ := config.Load(cfgPath)

	// Turn storage.path into a concrete file path and make sure the parent dir exists.
	dataPath := cfg.Storage.Path
	if dataPath == "" {
		dataPath = "/var/lib/logship"
	}
	// If it's a dir, use logship.db inside it
	if fi, err := os.Stat(dataPath); (err == nil && fi.IsDir()) || os.IsNotExist(err) {
		_ = os.MkdirAll(dataPath, 0o755)
		dataPath = filepath.Join(dataPath, "logship.db")
	} else {
		_ = os.MkdirAll(filepath.Dir(dataPath), 0o755)
	}

	db, err := store.Open(dataPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	log.Printf("store: using %s", dataPath)

	// ... start the rest of your app as before (ingest, forwarder, ui, etc.)
	_ = db
	select {}
}
