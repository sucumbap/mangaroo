package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sucumbap/mangaroo/internal/infrastructure/storage"
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
}

func (md *MangaDownloader) SetElasticClient(elasticClient *storage.ElasticClient) {
	md.elastic = elasticClient
	if md.elastic == nil {
		log.Println("Warning: Elasticsearch client is not set.")
	}
}

func NewMangaDownloader(config Config, mangaID string) (*MangaDownloader, error) {
	// Initialize Chrome
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(config.UserAgent),
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-zygote", true),
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.Flag("remote-debugging-address", "0.0.0.0"),
		chromedp.Flag("window-size", "1280,800"),
		chromedp.Flag("user-data-dir", os.Getenv("CHROMIUM_USER_DATA_DIR")),
	)

	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	return &MangaDownloader{
		config:    config,
		ctx:       ctx,
		cancelCtx: func() { cancelCtx(); cancelAlloc() },
		elastic:   nil,
		mangaID:   mangaID, // Set mangaID
	}, nil
}

func (md *MangaDownloader) Close() {
	md.cancelCtx()
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
	err := chromedp.Run(md.ctx,
		chromedp.Navigate(md.config.BaseURL),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.querySelectorAll('div.chapters table.uk-table tbody tr').length`, &count),
	)
	return count, err
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
	var imageURLs []string
	err := chromedp.Run(md.ctx,
		chromedp.Navigate(chapterURL),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('div#imgs img')).map(img => {
				return img.getAttribute('data-src') || img.getAttribute('src');
			}).filter(url => url && !url.startsWith('data:'))
		`, &imageURLs),
	)
	return imageURLs, err
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
	var status string
	err := chromedp.Run(md.ctx,
		chromedp.Navigate(md.config.BaseURL),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
            // Find the status element by its specific class structure
            const statusElement = document.querySelector('li.d-row-small div.status');
            
            // Return the status text if found, otherwise "Unknown"
            statusElement ? statusElement.textContent.trim() : "Unknown"
        `, &status),
	)

	// Clean up the status text
	if err == nil && status != "" {
		status = strings.TrimSpace(status)
		// Remove any "status" class text if it appears in the content
		status = strings.ReplaceAll(status, "status", "")
		status = strings.TrimSpace(status)
	}

	return status, err
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
	ext := determineFileExtension(url, resp.Header.Get("Content-Type"))

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
	var title string
	err := chromedp.Run(md.ctx,
		chromedp.Navigate(md.config.BaseURL),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
            // Try multiple selectors
            document.querySelector('h1.heading')?.textContent.trim() || 
            document.querySelector('h1.title')?.textContent.trim() || 
            document.querySelector('div.manga-info h1')?.textContent.trim() || 
            document.title.split('|')[0].trim() || 
            "unknown"
        `, &title),
	)
	return title, err
}

func determineFileExtension(url, contentType string) string {
	// Check content type first
	switch {
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		return "jpg"
	case strings.Contains(contentType, "png"):
		return "png"
	case strings.Contains(contentType, "webp"):
		return "webp"
	}

	// Fall back to URL extension
	parts := strings.Split(url, ".")
	if len(parts) > 1 {
		possibleExt := parts[len(parts)-1]
		if len(possibleExt) <= 4 { // Reasonable extension length
			return possibleExt
		}
	}

	return "jpg" // Default
}
