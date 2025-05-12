package utils

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type AppError struct {
	Error   error
	Message string
	Code    int
}

func LogError(context string, err error) {
	log.Printf("[ERROR] %s: %v", context, err)
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func HandleError(w http.ResponseWriter, appErr AppError) {
	if appErr.Error != nil {
		LogError(appErr.Message, appErr.Error)
	}
	RespondWithError(w, appErr.Code, appErr.Message)
}

func DetermineFileExtension(url, contentType string) string {
	// Check content type first
	switch {
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		return "jpg"
	case strings.Contains(contentType, "png"):
		return "png"
	case strings.Contains(contentType, "webp"):
		return "webp"
	}

	// Fall back to URL extension
	parts := strings.Split(url, ".")
	if len(parts) > 1 {
		possibleExt := parts[len(parts)-1]
		if len(possibleExt) <= 4 { // Reasonable extension length
			return possibleExt
		}
	}

	return "jpg" // Default
}
