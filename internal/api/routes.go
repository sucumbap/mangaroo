package api

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sucumbap/mangaroo/pkg/logger"
)

func NewRouter(h *MangaHandler, log logger.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Core middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(LoggingMiddleware(log))
	r.Use(RecoveryMiddleware(log))
	r.Use(middleware.Recoverer)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/manga/download", h.Download)
		r.Get("/health", h.HealthCheck)
	})

	return r
}
