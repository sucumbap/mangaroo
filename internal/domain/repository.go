package domain

import "context"

type MangaRepository interface {
	Store(manga *Manga) error
	FindByID(id string) (*Manga, error)
	Ping() error
}

type DownloaderService interface {
	DownloadManga(ctx context.Context, url string) (*Manga, error)
	DownloadChapter(ctx context.Context, mangaID string, number int) (*Chapter, error)
}
