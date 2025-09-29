package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

	"wakemap/internal/data"
	"wakemap/internal/server"
)

func getenvExpanded(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return os.ExpandEnv(v) // expands ${HOME} etc.
}

func main() {
	// Load .env if present; real env vars still win.
	_ = godotenv.Load(".env")

	port := getenvExpanded("PORT", "8080")
	addr := ":" + port

	// Default DB under repo’s ./devdata if unset.
	// If you put ${HOME} in .env it will expand here.
	dbPath := getenvExpanded("WAKEMAP_DB", filepath.Join(".", "devdata", "wakemap.db"))
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	store, err := data.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	api := &server.API{Store: store}

	mux := server.NewMux(api)

	log.Printf("wakemap dev server on http://localhost:%s  db=%s", port, dbPath)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
