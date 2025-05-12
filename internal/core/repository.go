package core

type MangaRepository interface {
	SaveManga(manga Manga) error
	GetMangaByID(id string) (Manga, error)
	GetAllManga() ([]Manga, error)
	DeleteManga(id string) error
}
