package storage

import (
	"fmt"

	"github.com/sucumbap/mangaroo/internal/core"
)

func (es *ElasticService) IndexMangaImage(indexName string, chapterID int, imageNum int, imagePath string, metadata map[string]interface{}) error {
	return es.ElasticClient.IndexMangaImage(indexName, chapterID, imageNum, imagePath, metadata)
}

func (es *ElasticService) SearchMangaImage(indexName string, query string) ([]string, error) {
	// Implement this using the ElasticClient
	// This is a stub implementation - you'll need to add actual search functionality
	return []string{}, fmt.Errorf("not implemented")
}

func (es *ElasticService) DeleteMangaImage(indexName string, chapterID int) error {
	// Implement using ElasticClient
	return fmt.Errorf("not implemented")
}

func (es *ElasticService) GetMangaImage(indexName string, chapterID int) ([]string, error) {
	// Implement using ElasticClient
	return []string{}, fmt.Errorf("not implemented")
}

// Add remaining interface methods with proper implementation
func (es *ElasticService) IndexManga(indexName string, manga core.Manga) error {
	// Implement
	return fmt.Errorf("not implemented")
}

func (es *ElasticService) GetManga(indexName string, mangaID string) (core.Manga, error) {
	// Implement
	return core.Manga{}, fmt.Errorf("not implemented")
}

func (es *ElasticService) GetAllManga(indexName string) ([]core.Manga, error) {
	// Implement
	return []core.Manga{}, fmt.Errorf("not implemented")
}

func (es *ElasticService) DeleteManga(indexName string, mangaID string) error {
	// Implement
	return fmt.Errorf("not implemented")
}

func (es *ElasticService) SearchManga(indexName string, query string) ([]core.Manga, error) {
	// Implement
	return []core.Manga{}, fmt.Errorf("not implemented")
}

func (es *ElasticService) GetMangaIndexName(mangaTitle string, mangaID string) string {
	return es.ElasticClient.GetMangaIndexName(mangaTitle, mangaID)
}

type ElasticService struct {
	ElasticClient *ElasticClient
}
type ElasticServiceInterface interface {
	IndexMangaImage(indexName string, chapterID int, imageNum int, imagePath string, metadata map[string]interface{}) error
	SearchMangaImage(indexName string, query string) ([]string, error)
	DeleteMangaImage(indexName string, chapterID int) error
	GetMangaImage(indexName string, chapterID int) ([]string, error)
	IndexManga(indexName string, manga core.Manga) error
	GetManga(indexName string, mangaID string) (core.Manga, error)
	GetAllManga(indexName string) ([]core.Manga, error)
	DeleteManga(indexName string, mangaID string) error
	SearchManga(indexName string, query string) ([]core.Manga, error)
	GetMangaIndexName(mangaTitle string, mangaID string) string
}

func NewElasticService(elasticClient *ElasticClient) *ElasticService {
	return &ElasticService{
		ElasticClient: elasticClient,
	}
}
