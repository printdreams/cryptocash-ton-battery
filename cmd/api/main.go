package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/printdreams/cryptocash-ton-battery/internal/config"
	"github.com/printdreams/cryptocash-ton-battery/internal/firebase"
	"github.com/printdreams/cryptocash-ton-battery/internal/handler"
)

// @title           Battery implementation for TON network
// @version         1.0
// @description     Backend service for the Cryptocash Battery on the TON network.
// @BasePath        /
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	fb, err := firebase.Init(ctx, cfg)
	if err != nil {
		log.Fatalf("firebase: %v", err)
	}
	log.Printf("Firebase initialized for project %q", cfg.ProjectID)

	r := chi.NewRouter()
	handler.SetupRoutes(r, handler.Deps{Firebase: fb})

	log.Printf("Server running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
