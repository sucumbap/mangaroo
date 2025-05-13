package storage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type ElasticClient struct {
	client *elasticsearch.Client
}

func NewElasticClient(address string) (*ElasticClient, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{address},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}
	return &ElasticClient{client: client}, nil
}

func (ec *ElasticClient) IndexMangaImage(indexName string, chapterID int, imageNum int, imagePath string, metadata map[string]interface{}) error {
	// Read the image file
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("failed to read image file: %w", err)
	}

	// Encode as Base64
	imgBase64 := base64.StdEncoding.EncodeToString(imgData)

	// Determine content type based on file extension
	contentType := "image/jpeg"
	if strings.HasSuffix(imagePath, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(imagePath, ".webp") {
		contentType = "image/webp"
	}

	doc := map[string]interface{}{
		"chapter_id":    chapterID,
		"image_num":     imageNum,
		"image_data":    imgBase64,
		"content_type":  contentType,
		"downloaded_at": time.Now().UTC(),
		"metadata":      metadata,
	}

	docJSON, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: fmt.Sprintf("%d-%d", chapterID, imageNum),
		Body:       strings.NewReader(string(docJSON)),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), ec.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	log.Printf("Successfully indexed image %d from chapter %d in index %s", imageNum, chapterID, indexName)
	return nil
}

func (ec *ElasticClient) Ping() error {
	req := esapi.PingRequest{}
	res, err := req.Do(context.Background(), ec.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ping failed: %s", res.String())
	}
	return nil
}

func (ec *ElasticClient) EnsureIndex(indexName string) error {
	res, err := ec.client.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		// Create index
		createRes, err := ec.client.Indices.Create(indexName)
		if err != nil {
			return err
		}
		defer createRes.Body.Close()

		if createRes.IsError() {
			return fmt.Errorf("failed to create index: %s", createRes.String())
		}
		log.Printf("Created index: %s", indexName)
	}
	return nil
}

func (ec *ElasticClient) GetMangaIndexName(mangaTitle, mangaID string) string {
	cleanTitle := strings.ToLower(strings.TrimSpace(mangaTitle))
	cleanTitle = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(cleanTitle, "_")
	cleanTitle = strings.Trim(cleanTitle, "_")
	return fmt.Sprintf("manga_%s_%s", cleanTitle, mangaID)

}
