package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", handler.HomeHandler)
		r.Post("/download", handler.DownloadPostHandler)
	})

	return r
}
