package storage

import (
	"context"
	"time"
)

// ArticleCard представляет обработанную карточку для сохранения в БД
type ArticleCard struct {
	CanonicalURL string
	Title        string
	Text         string
	ImageURL     string
	DateUTC      time.Time
	Language     string
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
}
