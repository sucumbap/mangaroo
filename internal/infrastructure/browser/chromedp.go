package browser

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sucumbap/mangaroo/internal/domain"
	"github.com/sucumbap/mangaroo/internal/infrastructure/http"
)

type ChromeDP struct {
	ctx       context.Context
	cancel    context.CancelFunc
	userAgent string
	timeout   time.Duration
}

func New(userAgent string, timeout time.Duration, headless bool) (*ChromeDP, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(userAgent),
		chromedp.Flag("headless", headless),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)

	return &ChromeDP{
		ctx:       ctx,
		cancel:    func() { cancelCtx(); cancelAlloc() },
		userAgent: userAgent,
		timeout:   timeout,
	}, nil
}

func (c *ChromeDP) Close() error {
	c.cancel()
	return nil
}

func (c *ChromeDP) GetTitle(url string) (string, error) {
	var title string
	err := c.run(url, `document.querySelector('h1.heading')?.textContent.trim()`, &title)
	return title, err
}

func (c *ChromeDP) run(url, js string, result interface{}) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(js, result),
	)
}

func (c *ChromeDP) DownloadManga(ctx context.Context, url string) (*domain.Manga, error) {
	manga := &domain.Manga{}
	// Get manga details
	title, err := c.GetTitle(url)
	if err != nil {
		return nil, err
	}
	manga.Title = title

	// Get total chapters
	totalChapters, err := c.getTotalChapters(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get total chapters: %w", err)
	}

	// Download each chapter
	for i := 1; i <= totalChapters; i++ {
		if _, err := c.DownloadChapter(ctx, manga.ID, i); err != nil {
			log.Printf("Error downloading chapter %d: %v", i, err)
		}
		time.Sleep(3 * time.Second) // Be polite to the server
	}

	return manga, nil
}

func (c *ChromeDP) DownloadChapter(ctx context.Context, mangaID string, number int) (*domain.Chapter, error) {
	chapter := &domain.Chapter{
		Number: number,
	}

	chapterURL := fmt.Sprintf("%s/c%d", mangaID, number)
	chapterFolder := filepath.Join("output", fmt.Sprintf("c%d", number))

	if err := os.MkdirAll(chapterFolder, 0755); err != nil {
		return nil, fmt.Errorf("error creating folder for chapter %d: %w", number, err)
	}

	imageURLs, err := c.getChapterImageURLs(chapterURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get image URLs: %w", err)
	}

	// Download images
	for i, imgURL := range imageURLs {
		imagePath := filepath.Join(chapterFolder, fmt.Sprintf("%03d.jpg", i+1))
		if err := c.downloadImage(imgURL, imagePath); err != nil {
			log.Printf("Error downloading image %d: %v", i+1, err)
		}
	}

	return chapter, nil
}
func (c *ChromeDP) getChapterImageURLs(chapterURL string) ([]string, error) {
	var imageURLs []string
	err := c.run(chapterURL, `
        Array.from(document.querySelectorAll('div#imgs img')).map(img => {
            return img.getAttribute('data-src') || img.getAttribute('src');
        }).filter(url => url && !url.startsWith('data:'))
    `, &imageURLs)
	return imageURLs, err
}

func (c *ChromeDP) downloadImage(url, path string) error {
	// Create a new HTTP client
	client := http.NewClient(c.userAgent, c.timeout)

	// Download the image
	imgData, contentType, err := client.DownloadImage(c.ctx, url, url) // Using URL as referer
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	// Determine file extension based on content type
	ext := ".jpg" // default
	switch {
	case strings.Contains(contentType, "png"):
		ext = ".png"
	case strings.Contains(contentType, "gif"):
		ext = ".gif"
	case strings.Contains(contentType, "webp"):
		ext = ".webp"
	}

	// Update path with correct extension
	path = strings.TrimSuffix(path, filepath.Ext(path)) + ext

	// Save the image to file
	if err := os.WriteFile(path, imgData, 0644); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	return nil
}

func (c *ChromeDP) getTotalChapters(url string) (int, error) {
	var count int
	err := c.run(url, `document.querySelectorAll('div.chapters table.uk-table tbody tr').length`, &count)
	return count, err
}
