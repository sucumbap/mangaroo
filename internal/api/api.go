package api

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func SetupRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", h.HomeHandler)
		r.Post("/download", h.DownloadPostHandler)

	})

	return r
}
