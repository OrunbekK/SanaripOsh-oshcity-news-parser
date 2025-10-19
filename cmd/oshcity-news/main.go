package main

import (
	"context"
	"log"
	"os"
	"oshcity-news-parser/internal/app"

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

	// Создаём парсер и фетчер
	f := fetcher.NewFetcher(cfg, logger)
	defer func() {
		if err := f.Close(); err != nil {
			logger.Error("Failed to close fetcher", "error", err.Error())
		}
	}()

	// Загружаем селекторы для RU
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

	// Создаём парсер (используем RU селекторы для примера)
	scr := scraper.NewScraper(selectorsRU, logger)
	dateParser := scraper.NewDateParser("ru")

	// Создаём оркестратор
	orchestrator := app.NewOrchestrator(cfg, logger, f, scr, dateParser)

	ctx := context.Background()

	// Запускаем пагинацию для RU
	logger.Info("Starting RU pagination")
	statsRU, err := orchestrator.Run(ctx, "ru")
	if err != nil {
		logger.Error("RU pagination failed", "error", err.Error())
	} else {
		logger.Info("RU pagination completed",
			"total_pages", statsRU.TotalPages,
			"total_cards", statsRU.TotalCards,
			"old_cards", statsRU.OldCards,
			"reason", statsRU.StoppedReason,
		)
	}

	logger.Info("Application finished")
}
