package storage

import (
	"github.com/sucumbap/mangaroo/internal/core"
)

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

func NewElasticService(client *ElasticClient) *ElasticService {
	return &ElasticService{ElasticClient: client}
}

func (es *ElasticService) IndexMangaImage(indexName string, chapterID int, imageNum int, imagePath string, metadata map[string]interface{}) error {
	return es.IndexMangaImage(indexName, chapterID, imageNum, imagePath, metadata)
}
func (es *ElasticService) SearchMangaImage(indexName string, query string) ([]string, error) {
	return es.SearchMangaImage(indexName, query)
}
func (es *ElasticService) DeleteMangaImage(indexName string, chapterID int) error {
	return es.DeleteMangaImage(indexName, chapterID)
}

func (es *ElasticService) GetMangaImage(indexName string, chapterID int) ([]string, error) {
	return es.GetMangaImage(indexName, chapterID)
}
