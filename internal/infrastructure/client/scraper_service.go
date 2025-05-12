package client

import (
	"fmt"
	"log"

	"github.com/sucumbap/mangaroo/internal/core"
)

type ScraperService struct {
	Downloader MangaDownloaderInterface
	Repository core.MangaRepository
}

func NewScraperService(downloader MangaDownloaderInterface, repository core.MangaRepository) *ScraperService {
	return &ScraperService{
		Downloader: downloader,
		Repository: repository,
	}
}

func (s *ScraperService) DownloadAndSaveManga(mangaID string) error {
	// Start the download process
	log.Printf("Starting download for manga ID: %s", mangaID)
	if err := s.Downloader.Run(); err != nil {
		return fmt.Errorf("failed to download manga: %w", err)
	}

	// Get manga metadata
	mangaTitle, err := s.Downloader.GetMangaTitle()
	if err != nil {
		log.Printf("Warning: Could not get manga title: %v", err)
		mangaTitle = "unknown"
	}

	mangaStatus, err := s.Downloader.GetMangaStatus()
	if err != nil {
		log.Printf("Warning: Could not get manga status: %v", err)
		mangaStatus = "unknown"
	}

	// Create a Manga object
	manga := core.Manga{
		ID:          mangaID,
		Title:       mangaTitle,
		Description: fmt.Sprintf("Status: %s", mangaStatus),
		Authors:     []string{},
		Genres:      []string{},
		CoverPath:   "",
		Chapters:    []core.Chapter{},
	}

	// Save manga metadata to the repository
	if err := s.Repository.SaveManga(manga); err != nil {
		return fmt.Errorf("failed to save manga metadata: %w", err)
	}

	log.Printf("Successfully downloaded and saved manga: %s", mangaTitle)
	return nil
}
