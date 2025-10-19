package storage

import (
	"context"
	"time"
)

// ArticleCard представляет обработанную карточку для сохранения в БД
type ArticleCard struct {
	CanonicalURL string // URL из Card
	Title        string
	Text         string
	ImageURL     string // ThumbnailURL из Card
	Date         time.Time
	Language     string
	SequenceNum  int    // Из Card
	CheckSum     string // SHA256 контента (256 символов)
}

// Repository интерфейс для работы с хранилищем карточек
type Repository interface {
	// UpsertCard сохраняет или обновляет карточку, возвращает (isNew, isUpdated, error)
	UpsertCard(ctx context.Context, card *ArticleCard) (isNew bool, isUpdated bool, err error)

	// ExistsByURL проверяет наличие карточки по URL
	ExistsByURL(ctx context.Context, url string) (bool, error)

	// GetLatestKnownDate получает последнюю загруженную дату для языка
	GetLatestKnownDate(ctx context.Context, lang string) (time.Time, error)

	// GetCardCount получает количество загруженных карточек для языка
	GetCardCount(ctx context.Context, lang string) (int, error)

	UpdateNewsCheckSum(ctx context.Context) (string, error)
}
