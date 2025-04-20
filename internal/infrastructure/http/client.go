// internal/infrastructure/http/client.go
package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseClient *http.Client
	userAgent  string
	timeout    time.Duration
}

func NewClient(userAgent string, timeout time.Duration) *Client {
	return &Client{
		baseClient: &http.Client{},
		userAgent:  userAgent,
		timeout:    timeout,
	}
}

func (c *Client) GetWithContext(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.baseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return buf.Bytes(), nil
}

func (c *Client) DownloadImage(ctx context.Context, url string, referer string) ([]byte, string, error) {
	headers := map[string]string{
		"Referer": referer,
	}

	data, err := c.GetWithContext(ctx, url, headers)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", err)
	}

	contentType := http.DetectContentType(data)
	return data, contentType, nil
}
