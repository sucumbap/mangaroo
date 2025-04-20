package domain

import "context"

type MangaService struct {
	repo       MangaRepository
	downloader DownloaderService
}

func NewMangaService(repo MangaRepository, downloader DownloaderService) *MangaService {
	return &MangaService{
		repo:       repo,
		downloader: downloader,
	}
}

func (s *MangaService) DownloadAndStore(ctx context.Context, url string) (*Manga, error) {
	manga, err := s.downloader.DownloadManga(ctx, url)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Store(manga); err != nil {
		return nil, err
	}

	return manga, nil
}

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

func (s *MangaService) HealthCheck(ctx context.Context) error {
	if err := s.repo.Ping(); err != nil {
		return err
	}

	return nil
}
