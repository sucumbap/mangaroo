package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sucumbap/mangaroo/internal/domain"
)

type ElasticStorage struct {
	client      *elasticsearch.Client
	indexPrefix string
}

func New(address, indexPrefix string) (*ElasticStorage, error) {
	cfg := elasticsearch.Config{Addresses: []string{address}}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return &ElasticStorage{client: client, indexPrefix: indexPrefix}, nil
}

func (es *ElasticStorage) Store(manga *domain.Manga) error {
	doc, _ := json.Marshal(manga)
	req := esapi.IndexRequest{
		Index:      es.getIndex(manga.ID),
		DocumentID: manga.ID,
		Body:       strings.NewReader(string(doc)),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elastic error: %s", res.String())
	}
	return nil
}

func (es *ElasticStorage) getIndex(id string) string {
	return fmt.Sprintf("%s%s", es.indexPrefix, id)
}

func (es *ElasticStorage) Ping() error {
	res, err := es.client.Ping()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ping failed: %s", res.String())
	}
	return nil
}

func (es *ElasticStorage) FindByID(id string) (*domain.Manga, error) {
	req := esapi.GetRequest{
		Index:      es.getIndex(id),
		DocumentID: id,
	}

	res, err := req.Do(context.Background(), es.client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("error getting document: %s", res.String())
	}

	var manga domain.Manga
	if err := json.NewDecoder(res.Body).Decode(&manga); err != nil {
		return nil, err
	}

	return &manga, nil
}
