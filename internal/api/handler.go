package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/sucumbap/mangaroo/internal/core"
	"github.com/sucumbap/mangaroo/internal/infrastructure/client"
	"github.com/sucumbap/mangaroo/internal/infrastructure/storage"
	"github.com/sucumbap/mangaroo/pkg/config"
)

type Handler struct {
	Config        *config.Config
	Repository    core.MangaRepository
	ElasticClient *storage.ElasticClient
}

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to Mangaroo!"))
}

func NewHandler(cfg *config.Config) (*Handler, error) {
	// Initialize ElasticClient
	elasticClient, err := storage.NewElasticClient(cfg.Elasticsearch.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Elasticsearch client: %w", err)
	}

	// Initialize Repository
	repository := storage.NewElasticMangaRepository(elasticClient, "mangaroo")

	return &Handler{
		Config:        cfg,
		Repository:    repository,
		ElasticClient: elasticClient,
	}, nil
}

func (h *Handler) DownloadPostHandler(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		log.Println("URL parameter is missing")
		http.Error(w, "URL parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("Download request received for URL: %s", urlParam)

	// Extract manga ID from URL
	var mangaID string
	if parts := strings.Split(urlParam, "/manga/"); len(parts) > 1 {
		mangaID = strings.TrimSuffix(parts[1], "/")
	} else {
		log.Println("Could not extract manga ID from URL, using 'unknown'")
		mangaID = "unknown"
	}

	// Test Elasticsearch connection
	log.Println("Testing Elasticsearch connection...")
	if err := h.ElasticClient.Ping(); err != nil {
		log.Printf("Elasticsearch ping failed: %v", err)
		http.Error(w, fmt.Sprintf("Elasticsearch not available: %v", err), http.StatusServiceUnavailable)
		return
	}
	log.Println("Elasticsearch connection successful")

	config := client.Config{
		BaseURL:      urlParam,
		OutputFolder: h.Config.Downloader.OutputFolder,
		UserAgent:    h.Config.Downloader.UserAgent,
	}

	var downloader *client.MangaDownloader
	var err error

	// Initialize downloader with manga ID
	log.Println("Initializing manga downloader...")
	downloader, err = client.NewMangaDownloader(config, mangaID)
	if err != nil {
		log.Printf("Failed to initialize downloader: %v", err)
		http.Error(w, fmt.Sprintf("Failed to initialize downloader: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		log.Println("Cleaning up resources...")
		if r := recover(); r != nil {
			log.Printf("PANIC RECOVERED in DownloadPostHandler: %v", r)
			debug.PrintStack() // Print stack trace for debugging
			http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
		}

		if downloader != nil {
			log.Println("Closing downloader...")
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("PANIC RECOVERED in downloader.Close(): %v", r)
					}
				}()
				downloader.Close()
			}()
			log.Println("Downloader closed")
		}
	}()

	// Set Elasticsearch client
	downloader.SetElasticClient(h.ElasticClient)

	// Get manga title
	mangaTitle, err := downloader.GetMangaTitle()
	if err != nil {
		log.Printf("Warning: Could not get manga title: %v", err)
		mangaTitle = "unknown"
	}

	// Create manga-specific index name
	indexName := h.ElasticClient.GetMangaIndexName(mangaTitle, mangaID)

	// Ensure index exists
	if err := h.ElasticClient.EnsureIndex(indexName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to ensure index exists: %v", err), http.StatusInternalServerError)
		return
	}

	// Start download process
	if err := downloader.Run(); err != nil {
		http.Error(w, fmt.Sprintf("Download failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Save manga metadata to repository
	manga := core.Manga{
		ID:    mangaID,
		Title: mangaTitle,
		// Add other metadata...
	}

	if err := h.Repository.SaveManga(manga); err != nil {
		log.Printf("Warning: Failed to save manga metadata: %v", err)
		// Continue anyway as we've already downloaded the images
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Download complete",
		"index":   indexName,
	})
}
