package app

import (
	"context"
	"fmt"
	"time"

	"oshcity-news-parser/internal/config"
	"oshcity-news-parser/internal/fetcher"
	"oshcity-news-parser/internal/observability"
	"oshcity-news-parser/internal/scraper"
)

type Orchestrator struct {
	cfg        *config.Config
	logger     *observability.Logger
	fetcher    *fetcher.Fetcher
	scraper    *scraper.Scraper
	dateParser *scraper.DateParser
}

func NewOrchestrator(
	cfg *config.Config,
	logger *observability.Logger,
	f *fetcher.Fetcher,
	s *scraper.Scraper,
	dp *scraper.DateParser,
) *Orchestrator {
	return &Orchestrator{
		cfg:        cfg,
		logger:     logger,
		fetcher:    f,
		scraper:    s,
		dateParser: dp,
	}
}

type PaginationStats struct {
	TotalPages          int
	TotalCards          int
	OldCards            int
	ConsecutiveOldPages int
	StoppedReason       string
}

// Run запускает пайплайн пагинации для языка
func (o *Orchestrator) Run(ctx context.Context, lang string) (*PaginationStats, error) {
	baseURL := o.getBaseURL(lang)
	if baseURL == "" {
		return nil, fmt.Errorf("no base URL for language: %s", lang)
	}

	maxPages := o.getMaxPages(lang)
	latestKnownDate := time.Now().UTC().AddDate(0, 0, -o.cfg.Pagination.DaysBackThreshold).Truncate(24 * time.Hour)

	o.logger.Info("Starting pagination",
		"language", lang,
		"base_url", baseURL,
		"max_pages", maxPages,
		"latest_known_date", latestKnownDate.Format("2006-01-02"),
		"stop_on_chain_pages", o.cfg.Pagination.StopOnKnownChainPages,
	)

	stats := &PaginationStats{}
	currentURL := baseURL
	consecutiveOldPages := 0

	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		o.logger.Info("Processing page",
			"language", lang,
			"page", pageNum,
			"url", currentURL,
		)

		// Фетчим страницу
		resp, err := o.fetcher.Fetch(ctx, currentURL, lang)
		if err != nil {
			o.logger.Error("Fetch failed",
				"language", lang,
				"page", pageNum,
				"url", currentURL,
				"error", err.Error(),
			)
			stats.StoppedReason = fmt.Sprintf("fetch error at page %d: %v", pageNum, err)
			return stats, err
		}

		// Парсим листинг
		cards, err := o.scraper.ParseListing(string(resp.Body))
		if err != nil {
			o.logger.Error("Parse listing failed",
				"language", lang,
				"page", pageNum,
				"error", err.Error(),
			)
			stats.StoppedReason = fmt.Sprintf("parse error at page %d: %v", pageNum, err)
			return stats, err
		}

		if len(cards) == 0 {
			o.logger.Info("No cards found on page",
				"language", lang,
				"page", pageNum,
			)
			stats.StoppedReason = fmt.Sprintf("no cards on page %d", pageNum)
			break
		}

		stats.TotalPages++
		stats.TotalCards += len(cards)

		// Проверяем, сколько карточек старые
		oldCardsOnPage := 0
		for i, card := range cards {
			cardDate, err := o.dateParser.Parse(card.DateRaw)
			if err != nil {
				o.logger.Warn("Failed to parse card date",
					"language", lang,
					"page", pageNum,
					"card_title", card.Title,
					"date_raw", card.DateRaw,
					"error", err.Error(),
				)
				// Считаем как не-старую (ошибка парсинга → новая)
				continue
			}

			// Debug: выводим информацию по каждой карточке
			o.logger.Debug("Card info",
				"language", lang,
				"page", pageNum,
				"card_num", i+1,
				"title", card.Title,
				"date", cardDate.Format("2006-01-02"),
				"url", card.URL,
				"thumbnail_url", card.ThumbnailURL,
			)

			if cardDate.Before(latestKnownDate) {
				oldCardsOnPage++
			}
		}

		stats.OldCards += oldCardsOnPage

		o.logger.Info("Page analysis",
			"language", lang,
			"page", pageNum,
			"total_cards", len(cards),
			"old_cards", oldCardsOnPage,
			"consecutive_old_pages", consecutiveOldPages,
		)

		// Проверяем, все ли карточки на странице старые
		if oldCardsOnPage == len(cards) {
			consecutiveOldPages++
			o.logger.Info("All cards on page are old",
				"language", lang,
				"page", pageNum,
				"consecutive_old_pages", consecutiveOldPages,
			)

			if consecutiveOldPages >= o.cfg.Pagination.StopOnKnownChainPages {
				o.logger.Info("Stopping: reached consecutive old pages threshold",
					"language", lang,
					"threshold", o.cfg.Pagination.StopOnKnownChainPages,
					"page", pageNum,
				)
				stats.StoppedReason = fmt.Sprintf("reached %d consecutive old pages at page %d", o.cfg.Pagination.StopOnKnownChainPages, pageNum)
				break
			}
		} else {
			// Сбрасываем счётчик, если нашли новые карточки
			consecutiveOldPages = 0
			o.logger.Info("Found new cards, reset consecutive counter",
				"language", lang,
				"page", pageNum,
			)
		}

		// Ищем ссылку на следующую страницу
		nextLink, err := o.scraper.FindNextPageLink(string(resp.Body))
		if err != nil {
			o.logger.Error("Failed to extract next link",
				"language", lang,
				"page", pageNum,
				"error", err.Error(),
			)
			stats.StoppedReason = fmt.Sprintf("failed to extract next link at page %d: %v", pageNum, err)
			break
		}

		if nextLink == "" {
			o.logger.Info("No next link found",
				"language", lang,
				"page", pageNum,
			)
			stats.StoppedReason = fmt.Sprintf("no next link at page %d", pageNum)
			break
		}

		currentURL = nextLink
		o.logger.Debug("Next URL extracted",
			"language", lang,
			"page", pageNum,
			"next_url", nextLink,
		)
	}

	o.logger.Info("Pagination completed",
		"language", lang,
		"total_pages", stats.TotalPages,
		"total_cards", stats.TotalCards,
		"old_cards", stats.OldCards,
		"reason", stats.StoppedReason,
	)

	return stats, nil
}

func (o *Orchestrator) getBaseURL(lang string) string {
	if lang == "ky" {
		return o.cfg.BaseURLs.KY
	}
	return o.cfg.BaseURLs.RU
}

func (o *Orchestrator) getMaxPages(lang string) int {
	if lang == "ky" {
		return o.cfg.Pagination.MaxPagesKY
	}
	return o.cfg.Pagination.MaxPagesRU
}
