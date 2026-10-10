package main

import (
	"log"
	"net/http"

	"github.com/nexkod/nexedar/internal/config"
	"github.com/nexkod/nexedar/internal/db"
	"github.com/nexkod/nexedar/internal/handlers"
	"github.com/nexkod/nexedar/internal/storage"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting NexEdar server on port %s...", cfg.Port)

	database, err := db.Init(cfg.DataDir)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.Close()

	store, err := storage.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("Storage initialization failed: %v", err)
	}

	// Seed default projects if empty
	_ = database.UpsertProject(db.Project{
		ID:          "nexronda",
		Name:        "NexRonda",
		Description: "Real-time convoy riding & low-latency intercom app for motorcyclists.",
	})
	_ = database.UpsertProject(db.Project{
		ID:          "nexdagang",
		Name:        "NexDagang",
		Description: "Next-gen POS & store management companion for retail businesses.",
	})

	srv := handlers.NewServer(cfg, database, store)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Printf("NexEdar listening on :%s (Public URL: %s)", cfg.Port, cfg.PublicURL)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
