package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"oshcity-news-parser/internal/config"
	"oshcity-news-parser/internal/fetcher"
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

	// Загружаем селекторы
	selectorsRU := &scraper.Selectors{
		ListContainer:  "div.elementor-posts-container",
		CardSelectors:  "article.elementor-post",
		TitleSelectors: []string{"h3.elementor-post__title > a", ".elementor-post__title > a"},
		URLSelectors:   []string{"h3.elementor-post__title > a", ".elementor-post__read-more"},
		DateSelectors:  []string{"span.elementor-post-date", ".elementor-post__meta-data span"},
		NextPageLink:   []string{"a.next", "a[rel='next']"},
	}

	// Создаём парсер и фетчер
	f := fetcher.NewFetcher(cfg)
	scr := scraper.NewScraper(selectorsRU)

	// Фетчим первую страницу
	ctx := context.Background()
	resp, err := f.Fetch(ctx, cfg.BaseURLs.RU, "ru")
	if err != nil {
		log.Fatalf("Fetch failed: %v", err)
	}

	fmt.Printf("✓ Fetched %d bytes\n", len(resp.Body))

	// Парсим листинг
	cards, err := scr.ParseListing(string(resp.Body))
	if err != nil {
		log.Fatalf("Parse failed: %v", err)
	}

	fmt.Printf("✓ Found %d cards\n\n", len(cards))

	// Выводим первые 3 карточки
	for i, card := range cards {
		if i >= 3 {
			break
		}
		fmt.Printf("[%d] Title: %s\n", i+1, card.Title)
		fmt.Printf("    URL: %s\n", card.URL)
		fmt.Printf("    Date: %s\n", card.DateRaw)
		fmt.Printf("\n")
	}
}
