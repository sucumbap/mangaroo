package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sucumbap/mangaroo/internal/core"
)

type ElasticMangaRepository struct {
	elasticClient *ElasticClient
	indexPrefix   string
}

func NewElasticMangaRepository(client *ElasticClient, indexPrefix string) *ElasticMangaRepository {
	return &ElasticMangaRepository{
		elasticClient: client,
		indexPrefix:   indexPrefix,
	}
}

func (r *ElasticMangaRepository) SaveManga(manga core.Manga) error {
	indexName := fmt.Sprintf("%s_manga", r.indexPrefix)

	// Ensure index exists
	if err := r.elasticClient.EnsureIndex(indexName); err != nil {
		return fmt.Errorf("failed to ensure index exists: %w", err)
	}

	// Marshal manga to JSON
	docJSON, err := json.Marshal(manga)
	if err != nil {
		return fmt.Errorf("failed to marshal manga: %w", err)
	}

	// Index document
	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: manga.ID,
		Body:       strings.NewReader(string(docJSON)),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), r.elasticClient.client)
	if err != nil {
		return fmt.Errorf("failed to index manga: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	return nil
}

func (r *ElasticMangaRepository) GetMangaByID(id string) (core.Manga, error) {
	indexName := fmt.Sprintf("%s_manga", r.indexPrefix)

	req := esapi.GetRequest{
		Index:      indexName,
		DocumentID: id,
	}

	res, err := req.Do(context.Background(), r.elasticClient.client)
	if err != nil {
		return core.Manga{}, fmt.Errorf("failed to get manga: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return core.Manga{}, fmt.Errorf("manga not found")
	}

	if res.IsError() {
		return core.Manga{}, fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return core.Manga{}, fmt.Errorf("error parsing response: %w", err)
	}

	source, ok := result["_source"].(map[string]interface{})
	if !ok {
		return core.Manga{}, fmt.Errorf("unexpected response format")
	}

	var manga core.Manga
	sourceBytes, err := json.Marshal(source)
	if err != nil {
		return core.Manga{}, fmt.Errorf("error re-marshaling source: %w", err)
	}

	if err := json.Unmarshal(sourceBytes, &manga); err != nil {
		return core.Manga{}, fmt.Errorf("error unmarshaling manga: %w", err)
	}

	return manga, nil
}

func (r *ElasticMangaRepository) GetAllManga() ([]core.Manga, error) {
	indexName := fmt.Sprintf("%s_manga", r.indexPrefix)
	req := esapi.SearchRequest{
		Index: []string{indexName},
		Body:  strings.NewReader(`{"query": {"match_all": {}}}`),
	}
	res, err := req.Do(context.Background(), r.elasticClient.client)
	if err != nil {
		return nil, fmt.Errorf("failed to search manga: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch error: %s", res.String())
	}
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	hits, ok := result["hits"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}
	hitList, ok := hits["hits"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}
	var mangas []core.Manga
	for _, hit := range hitList {
		source, ok := hit.(map[string]interface{})["_source"].(map[string]interface{})
		if !ok {
			continue
		}
		sourceBytes, err := json.Marshal(source)
		if err != nil {
			return nil, fmt.Errorf("error re-marshaling source: %w", err)
		}
		var manga core.Manga
		if err := json.Unmarshal(sourceBytes, &manga); err != nil {
			return nil, fmt.Errorf("error unmarshaling manga: %w", err)
		}
		mangas = append(mangas, manga)
	}
	return mangas, nil
}

func (r *ElasticMangaRepository) DeleteManga(id string) error {
	indexName := fmt.Sprintf("%s_manga", r.indexPrefix)

	req := esapi.DeleteRequest{
		Index:      indexName,
		DocumentID: id,
	}

	res, err := req.Do(context.Background(), r.elasticClient.client)
	if err != nil {
		return fmt.Errorf("failed to delete manga: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	return nil
}
