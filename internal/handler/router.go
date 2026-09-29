package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "github.com/printdreams/cryptocash-ton-battery/docs"
	"github.com/printdreams/cryptocash-ton-battery/internal/firebase"
	swagFiles "github.com/swaggo/http-swagger"
)

type Deps struct {
	Firebase *firebase.Clients
}

func SetupRoutes(r *chi.Mux, d Deps) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/swagger/*", swagFiles.Handler())
	r.Get("/health", HealthCheck)

	fbHandler := NewFirebaseHandler(d.Firebase)
	r.Get("/firebase/status", fbHandler.Status)
}
