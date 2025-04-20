package domain

import "time"

type Manga struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Authors     []string  `json:"authors"`
	Genres      []string  `json:"genres"`
	Status      string    `json:"status"`
	CoverPath   string    `json:"cover_path,omitempty"`
	Chapters    []Chapter `json:"chapters"`
	CreatedAt   time.Time `json:"created_at"`
}

type Chapter struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Number   int    `json:"number"`
	Pages    []Page `json:"pages"`
	Uploaded string `json:"uploaded"`
}

type Page struct {
	Number      int    `json:"number"`
	Path        string `json:"path,omitempty"`
	ImageData   string `json:"image_data,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}
