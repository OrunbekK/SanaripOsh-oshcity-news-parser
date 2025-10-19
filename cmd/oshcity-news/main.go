package main

import (
	"context"
	"log"
	"os"

	"oshcity-news-parser/internal/config"
	"oshcity-news-parser/internal/fetcher"
	"oshcity-news-parser/internal/observability"
	"oshcity-news-parser/internal/scraper"
)

func main() {
	configPath := "configs/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализируем logger
	logger := observability.NewLogger(cfg.Observability.LogPath, cfg.Observability.LogLevel)
	logger.Info("Application started", "config", configPath)

	// Загружаем селекторы
	selectorsRU := &scraper.Selectors{
		ListContainer:  "div.elementor-posts-container",
		CardSelectors:  "article.elementor-post",
		TitleSelectors: []string{"h3.elementor-post__title > a", ".elementor-post__title > a"},
		URLSelectors:   []string{"h3.elementor-post__title > a@href", ".elementor-post__read-more@href"},
		ImageSelectors: []string{".elementor-post__thumbnail img@src", "img.attachment-medium_large@src"},
		TextSelectors:  []string{".elementor-post__excerpt p", ".elementor-post__excerpt"},
		DateSelectors:  []string{"span.elementor-post-date", ".elementor-post__meta-data span"},
		NextPageLink:   []string{"nav.elementor-pagination a.next@href", "a.page-numbers.next@href"},
	}

	// Создаём парсер и фетчер
	f := fetcher.NewFetcher(cfg, logger)
	scr := scraper.NewScraper(selectorsRU, logger)

	logger.Info("Fetching news", "url", cfg.BaseURLs.RU, "language", "ru")

	// Фетчим первую страницу
	ctx := context.Background()
	resp, err := f.Fetch(ctx, cfg.BaseURLs.RU, "ru")
	if err != nil {
		logger.Error("Fetch failed", "error", err.Error())
		log.Fatalf("Fetch failed: %v", err)
	}

	logger.Info("Fetched successfully", "size", len(resp.Body))

	// Парсим листинг
	cards, err := scr.ParseListing(string(resp.Body))
	if err != nil {
		logger.Error("Parse failed", "error", err.Error())
		log.Fatalf("Parse failed: %v", err)
	}

	logger.Info("Parsing completed", "cards-found", len(cards))

	// Выводим первые 3 карточки
	for i, card := range cards {
		if i >= 3 {
			break
		}

		logger.Info("Processing card",
			"num", i+1,
			"title", card.Title,
			"url", card.URL,
			"thumbnail_url", card.ThumbnailURL,
			"date", card.DateRaw,
			"text", card.Text[:min(50, len(card.Text))]+"...",
		)
	}

	logger.Info("Application finished")
}
