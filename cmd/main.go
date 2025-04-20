package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sucumbap/mangaroo/internal/api"
	"github.com/sucumbap/mangaroo/internal/domain"
	"github.com/sucumbap/mangaroo/internal/infrastructure/browser"
	"github.com/sucumbap/mangaroo/internal/infrastructure/storage"
	"github.com/sucumbap/mangaroo/pkg/config"
	"github.com/sucumbap/mangaroo/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	log := logger.New()

	// Setup infrastructure
	es, err := storage.New(cfg.Elasticsearch.URL, cfg.Elasticsearch.IndexPrefix)
	if err != nil {
		log.Error("Failed to create Elasticsearch client: %v", err)
		os.Exit(1)
	}

	chrome, err := browser.New(cfg.Browser.UserAgent, cfg.Browser.Timeout, cfg.Browser.Headless)
	if err != nil {
		log.Error("Failed to create browser client: %v", err)
		os.Exit(1)
	}
	defer chrome.Close()

	// Setup domain services
	mangaService := domain.NewMangaService(es, chrome)

	// Setup API
	handler := api.NewMangaHandler(*mangaService, log)
	router := api.NewRouter(handler, log)

	// Start server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed: %v", err)
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server shutdown error: %v", err)
	}

	log.Info("Server exited gracefully")
}
