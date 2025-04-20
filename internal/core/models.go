package core

type Manga struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Authors     []string  `json:"authors"`
	Genres      []string  `json:"genres"`
	CoverPath   string    `json:"cover_path"`
	Chapters    []Chapter `json:"chapters"`
}

type Chapter struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Number   string `json:"number"`
	Pages    []Page `json:"pages"`
	Uploaded string `json:"uploaded"`
}

type Page struct {
	Number int    `json:"number"`
	Path   string `json:"path"`
}
