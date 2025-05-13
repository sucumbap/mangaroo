package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sucumbap/mangaroo/internal/infrastructure/browser"
	"github.com/sucumbap/mangaroo/internal/infrastructure/storage"
	"github.com/sucumbap/mangaroo/internal/utils"
)

type Config struct {
	BaseURL      string
	OutputFolder string
	UserAgent    string
}

type MangaDownloader struct {
	config    Config
	ctx       context.Context
	cancelCtx context.CancelFunc
	elastic   *storage.ElasticClient
	mangaID   string
	browserDP browser.ChromeDPInterface
}
type MangaDownloaderInterface interface {
	GetMangaStatus() (string, error)
	GetMangaTitle() (string, error)
	downloadChapter(chapterNum int) error
	downloadAndDetermineExtension(url, tempPath string) (string, error)
	normalizeImageURL(url string) string
	getChapterImageURLs(chapterURL string) ([]string, error)
	getTotalChapters() (int, error)
	Run() error
	Close()
	SetElasticClient(elasticClient *storage.ElasticClient)
}

func (md *MangaDownloader) SetElasticClient(elasticClient *storage.ElasticClient) {
	md.elastic = elasticClient
	if md.elastic == nil {
		log.Println("Warning: Elasticsearch client is not set.")
	}
}

func NewMangaDownloader(config Config, mangaID string) (*MangaDownloader, error) {
	// Initialize ChromeDP context
	var browserDP browser.ChromeDPInterface = &browser.ChromeDP{}
	chromeDPctx, err := browserDP.InitChromeDP()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ChromeDP: %w", err)
	}

	return &MangaDownloader{
		config:    config,
		ctx:       chromeDPctx.Ctx,
		cancelCtx: func() { chromeDPctx.CancelCtx(); chromeDPctx.CancelAlloc() },
		elastic:   nil,
		mangaID:   mangaID,
		browserDP: browserDP,
	}, nil
}

func (md *MangaDownloader) Close() {
	if md == nil {
		return
	}

	if md.browserDP != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic when closing browserDP: %v", r)
				}
			}()
			md.browserDP.CloseChromeDP()
		}()
	}
}

func (md *MangaDownloader) Run() error {
	// Create output folder
	if err := os.MkdirAll(md.config.OutputFolder, 0755); err != nil {
		return fmt.Errorf("error creating output folder: %w", err)
	}
	// Get manga status first
	status, err := md.GetMangaStatus()
	if err != nil {
		log.Printf("Warning: Could not get manga status: %v", err)
		status = "Unknown"
	}
	fmt.Printf("Manga Status: %s\n", status)

	// Get total chapters
	totalChapters, err := md.getTotalChapters()
	if err != nil {
		return fmt.Errorf("failed to get chapter count: %w", err)
	}

	fmt.Printf("Found %d chapters\n", totalChapters)

	// Download each chapter
	for i := 1; i <= totalChapters; i++ {
		if err := md.downloadChapter(i); err != nil {
			log.Printf("Error downloading chapter %d: %v", i, err)
		}
		time.Sleep(3 * time.Second) // Be polite to the server
	}

	return nil
}

func (md *MangaDownloader) getTotalChapters() (int, error) {
	var count int

	if err := md.browserDP.Navigate(md.config.BaseURL); err != nil {
		return 0, fmt.Errorf("failed to navigate to URL: %w", err)
	}
	// Sleep for 2 seconds
	md.browserDP.Bsleep(2)
	// Evaluate the JavaScript to get the total chapter count
	result, err := md.browserDP.Evaluate(`document.querySelectorAll('div.chapters table.uk-table tbody tr').length`)
	if err != nil {
		return 0, fmt.Errorf("failed to evaluate JavaScript: %w", err)
	}
	count, err = strconv.Atoi(result)
	if err != nil {
		return 0, fmt.Errorf("failed to convert result to integer: %w", err)
	}

	return count, nil
}

func (md *MangaDownloader) downloadChapter(chapterNum int) error {
	chapterURL := fmt.Sprintf("%s/c%d", md.config.BaseURL, chapterNum)
	chapterFolder := filepath.Join(md.config.OutputFolder, fmt.Sprintf("c%d", chapterNum))

	if err := os.MkdirAll(chapterFolder, 0755); err != nil {
		return fmt.Errorf("error creating folder for chapter %d: %w", chapterNum, err)
	}

	imageURLs, err := md.getChapterImageURLs(chapterURL)
	if err != nil {
		return fmt.Errorf("failed to get image URLs: %w", err)
	}

	// Get manga title
	mangaTitle, err := md.GetMangaTitle()
	if err != nil {
		log.Printf("Warning: Could not get manga title: %v", err)
		mangaTitle = "unknown"
	}

	// Download all images first
	var downloadedImages []string
	for i, imgURL := range imageURLs {
		absURL := md.normalizeImageURL(imgURL)

		// First download to determine the file type
		tempPath := filepath.Join(chapterFolder, fmt.Sprintf("%03d_temp", i+1))
		ext, err := md.downloadAndDetermineExtension(absURL, tempPath)
		if err != nil {
			log.Printf("Error downloading image %d: %v", i+1, err)
			continue
		}

		// Now create the final file with correct extension
		finalPath := filepath.Join(chapterFolder, fmt.Sprintf("%03d.%s", i+1, ext))
		if err := os.Rename(tempPath, finalPath); err != nil {
			log.Printf("Error renaming temp file for image %d: %v", i+1, err)
			continue
		}

		downloadedImages = append(downloadedImages, finalPath)
		time.Sleep(500 * time.Millisecond)
	}

	// Upload to Elasticsearch
	if md.elastic != nil {
		indexName := md.elastic.GetMangaIndexName(mangaTitle, md.mangaID)
		for i, imgPath := range downloadedImages {
			metadata := map[string]interface{}{
				"manga_url":   md.config.BaseURL,
				"manga_title": mangaTitle,
				"manga_id":    md.mangaID,
				"chapter_num": chapterNum,
				"image_index": i + 1,
			}

			if err := md.elastic.IndexMangaImage(indexName, chapterNum, i+1, imgPath, metadata); err != nil {
				log.Printf("Failed to upload image %d to Elasticsearch: %v", i+1, err)
				continue // Skip deletion if upload failed
			}

			// Only delete if upload succeeded
			if err := os.Remove(imgPath); err != nil {
				log.Printf("Failed to delete image %s: %v", imgPath, err)
			}
		}

		// Delete chapter folder after all images are processed
		if err := os.RemoveAll(chapterFolder); err != nil {
			log.Printf("Failed to delete chapter folder %s: %v", chapterFolder, err)
		}
	}

	return nil
}
func (md *MangaDownloader) getChapterImageURLs(chapterURL string) ([]string, error) {
	// Navigate to the chapter URL
	if err := md.browserDP.Navigate(chapterURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to chapter URL: %w", err)
	}

	// Sleep for 3 seconds to allow the page to load
	md.browserDP.Bsleep(3)

	// Evaluate JavaScript to extract image URLs
	result, err := md.browserDP.Evaluate(`
        JSON.stringify(
            Array.from(document.querySelectorAll('div#imgs img')).map(img => {
                return img.getAttribute('data-src') || img.getAttribute('src');
            }).filter(url => url && !url.startsWith('data:'))
        )
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate JavaScript for image URLs: %w", err)
	}

	// Parse the JSON string into a slice of strings
	var imageURLs []string
	if err := json.Unmarshal([]byte(result), &imageURLs); err != nil {
		return nil, fmt.Errorf("failed to parse image URLs: %w", err)
	}

	return imageURLs, nil
}

func (md *MangaDownloader) normalizeImageURL(url string) string {
	if strings.HasPrefix(url, "http") {
		return url
	}

	if strings.HasPrefix(url, "//") {
		return "https:" + url
	}

	if !strings.HasPrefix(url, "/") {
		return "https://" + url
	}

	return "https://mangakatana.com" + url
}

func (md *MangaDownloader) GetMangaStatus() (string, error) {
	// Navigate to the base URL
	if err := md.browserDP.Navigate(md.config.BaseURL); err != nil {
		return "", fmt.Errorf("failed to navigate to base URL: %w", err)
	}

	// Sleep for 2 seconds to allow the page to load
	md.browserDP.Bsleep(2)

	// Evaluate JavaScript to extract the manga status
	result, err := md.browserDP.Evaluate(`
        // Find the status element by its specific class structure
        const statusElement = document.querySelector('li.d-row-small div.status');
        
        // Return the status text if found, otherwise "Unknown"
        statusElement ? statusElement.textContent.trim() : "Unknown"
    `)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate JavaScript for manga status: %w", err)
	}

	// Clean up the status text
	status := strings.TrimSpace(result)
	status = strings.ReplaceAll(status, "status", "")
	status = strings.TrimSpace(status)

	return status, nil
}

func (md *MangaDownloader) downloadAndDetermineExtension(url, tempPath string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", md.config.UserAgent)
	req.Header.Set("Referer", md.config.BaseURL)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Create temp file
	file, err := os.Create(tempPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Copy content while also keeping it in memory for Content-Type checking
	var buf bytes.Buffer
	multiWriter := io.MultiWriter(file, &buf)

	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		return "", err
	}

	// Determine extension
	ext := utils.DetermineFileExtension(url, resp.Header.Get("Content-Type"))

	// Additional verification for cases where Content-Type might be misleading
	if ext == "jpg" || ext == "jpeg" {
		// Quick check if it's actually a PNG
		if bytes.HasPrefix(buf.Bytes(), []byte("\x89PNG\r\n\x1a\n")) {
			ext = "png"
		}
	}

	return ext, nil
}

func (md *MangaDownloader) GetMangaTitle() (string, error) {
	if md.browserDP == nil {
		return "unknown", fmt.Errorf("browserDP is not initialized")
	}

	// Navigate to the base URL
	log.Printf("Getting manga title for: %s", md.config.BaseURL)
	if err := md.browserDP.Navigate(md.config.BaseURL); err != nil {
		log.Printf("Failed to navigate to base URL: %v", err)
		return "unknown", fmt.Errorf("failed to navigate to base URL: %w", err)
	}

	// Sleep for 2 seconds to allow the page to load
	log.Println("Waiting for page to load...")
	if err := md.browserDP.Bsleep(2); err != nil {
		log.Printf("Sleep failed: %v", err)
		return "unknown", err
	}

	// Evaluate JavaScript to extract the manga title
	log.Println("Evaluating JavaScript to extract title...")
	result, err := md.browserDP.Evaluate(`
        // Try multiple selectors
        document.querySelector('h1.heading')?.textContent.trim() || 
        document.querySelector('h1.title')?.textContent.trim() || 
        document.querySelector('div.manga-info h1')?.textContent.trim() || 
        document.title.split('|')[0].trim() || 
        "unknown"
    `)
	if err != nil {
		log.Printf("Failed to evaluate JavaScript for manga title: %v", err)
		return "unknown", fmt.Errorf("failed to evaluate JavaScript for manga title: %w", err)
	}

	log.Printf("Found manga title: %s", result)
	return strings.TrimSpace(result), nil
}
