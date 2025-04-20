package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/sucumbap/mangaroo/internal/domain"
	"github.com/sucumbap/mangaroo/pkg/logger"
)

type MangaHandler struct {
	service domain.MangaService
	log     logger.Logger
}

func NewMangaHandler(service domain.MangaService, log logger.Logger) *MangaHandler {
	return &MangaHandler{service: service, log: log}
}

func (h *MangaHandler) Download(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		respondError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	manga, err := h.service.DownloadAndStore(r.Context(), url)
	if err != nil {
		h.log.Error("Download failed: %v", err)

		// More specific error handling
		switch {
		case strings.Contains(err.Error(), "not found"):
			respondError(w, http.StatusNotFound, "Manga not found")
		case strings.Contains(err.Error(), "timeout"):
			respondError(w, http.StatusGatewayTimeout, "Request timeout")
		default:
			respondError(w, http.StatusInternalServerError, "Failed to download manga")
		}
		return
	}

	respondSuccess(w, manga)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data == nil {
		return
	}

	// Use json.MarshalIndent for development (prettier output)
	// In production, use json.Marshal for better performance
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal JSON response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
		return
	}

	if _, err := w.Write(jsonData); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, map[string]string{
		"error": message,
	})
}

func respondSuccess(w http.ResponseWriter, data interface{}) {
	response := map[string]interface{}{
		"status": "success",
		"data":   data,
	}
	respondJSON(w, http.StatusOK, response)
}

func (h *MangaHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.service.HealthCheck(r.Context()); err != nil {
		h.log.Error("Health check failed: %v", err)
		respondError(w, http.StatusServiceUnavailable, "Service unavailable")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
