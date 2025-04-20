package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/sucumbap/mangaroo/internal/infrastructure"
	"github.com/sucumbap/mangaroo/internal/infrastructure/storage"
	"github.com/sucumbap/mangaroo/internal/utils"
	"github.com/sucumbap/mangaroo/pkg/config"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Mangaroo!"))
}

func DownloadPostHandler(w http.ResponseWriter, r *http.Request) {

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load configuration: %v", err), http.StatusInternalServerError)
		return
	}

	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	// Extract manga ID from URL
	var mangaID string
	if parts := strings.Split(urlParam, "/manga/"); len(parts) > 1 {
		mangaID = strings.TrimSuffix(parts[1], "/")
	} else {
		mangaID = "unknown"
	}

	// Initialize Elastic client
	elasticClient, err := storage.NewElasticClient(cfg.Elasticsearch.URL)
	if err != nil {
		utils.LogError("Elasticsearch connection", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to initialize Elasticsearch client")
		return
	}

	// Test Elasticsearch connection
	if err := elasticClient.Ping(); err != nil {
		log.Printf("Elasticsearch ping failed: %v", err)
		http.Error(w, fmt.Sprintf("Elasticsearch not available: %v", err), http.StatusServiceUnavailable)
		return
	}

	config := infrastructure.Config{
		BaseURL:      urlParam,
		OutputFolder: "output",
		UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	// Initialize downloader with manga ID
	downloader, err := infrastructure.NewMangaDownloader(config, mangaID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to initialize downloader: %v", err), http.StatusInternalServerError)
		return
	}
	defer downloader.Close()

	// Get manga title
	mangaTitle, err := downloader.GetMangaTitle()
	if err != nil {
		log.Printf("Warning: Could not get manga title: %v", err)
		mangaTitle = "unknown"
	}

	// Create manga-specific index name
	indexName := elasticClient.GetMangaIndexName(mangaTitle, mangaID)

	// Ensure index exists
	if err := elasticClient.EnsureIndex(indexName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to ensure index exists: %v", err), http.StatusInternalServerError)
		return
	}

	// Set Elasticsearch client
	downloader.SetElasticClient(elasticClient)

	// Start download process
	if err := downloader.Run(); err != nil {
		http.Error(w, fmt.Sprintf("Download failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Download complete",
		"index":   indexName, // Return the index name for reference
	})
}
