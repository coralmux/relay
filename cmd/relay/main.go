package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openclaw/openclaw-relay/internal/hub"
	"github.com/openclaw/openclaw-relay/internal/server"
	"github.com/openclaw/openclaw-relay/internal/store"
)

func main() {
	addr := flag.String("addr", ":8443", "Listen address")
	domain := flag.String("domain", "", "TLS domain (empty = no TLS)")
	dbPath := flag.String("db", "relay.db", "SQLite database path")
	adminKey := flag.String("admin-key", os.Getenv("RELAY_ADMIN_KEY"), "Admin API key")
	flag.Parse()

	db, err := store.NewSQLiteStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	h := hub.NewHub(db)

	srv := server.New(h, server.Config{
		Addr:      *addr,
		TLSDomain: *domain,
		AdminKey:  *adminKey,
		DBPath:    *dbPath,
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	log.Printf("Relay server running on %s", *addr)

	<-stop
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
