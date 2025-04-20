package utils

import (
	"encoding/json"
	"log"
	"net/http"
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
