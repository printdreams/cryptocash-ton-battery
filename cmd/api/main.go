package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/config"
	"github.com/printdreams/cryptocash-ton-battery/internal/firebase"
	"github.com/printdreams/cryptocash-ton-battery/internal/handler"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
	"github.com/printdreams/cryptocash-ton-battery/internal/nonce"
	"github.com/printdreams/cryptocash-ton-battery/internal/tonproof"
	"github.com/printdreams/cryptocash-ton-battery/internal/user"
)

// @title           Battery implementation for TON network
// @version         1.0
// @description     Backend service for the Cryptocash Battery on the TON network.
// @BasePath        /
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @securityDefinitions.apikey  AdminToken
// @in                          header
// @name                        X-Admin-Token
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

	nonceStore := nonce.NewStore(fb.Firestore, time.Duration(cfg.NonceTTLSeconds)*time.Second)
	userStore := user.NewStore(fb.Firestore)
	ledgerStore := ledger.NewStore(fb.Firestore)

	verifier := tonproof.Config{
		AllowedDomains:    cfg.AllowedDomains,
		ProofValidSeconds: cfg.ProofValidSeconds,
		NetworkGlobalID:   int32(cfg.NetworkGlobalID),
	}

	jwtIssuer := auth.NewIssuer(cfg.JWTSecret, time.Duration(cfg.JWTTTLSeconds)*time.Second)

	r := chi.NewRouter()
	handler.SetupRoutes(r, handler.Deps{
		Firebase:   fb,
		Nonces:     nonceStore,
		Verifier:   verifier,
		JWT:        jwtIssuer,
		Users:      userStore,
		Ledger:     ledgerStore,
		AdminToken: cfg.AdminToken,
		DevSign:    cfg.DevSignEnabled,
	})

	log.Printf("Server running on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
