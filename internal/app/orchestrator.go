package app

import (
	"context"
	"fmt"
	"oshcity-news-parser/internal/checksum"
	"oshcity-news-parser/internal/storage"
	"time"

	"oshcity-news-parser/internal/config"
	"oshcity-news-parser/internal/fetcher"
	"oshcity-news-parser/internal/observability"
	"oshcity-news-parser/internal/scraper"
)

type Orchestrator struct {
	cfg         *config.Config
	logger      *observability.Logger
	fetcher     *fetcher.Fetcher
	scraper     *scraper.Scraper
	dateParser  *scraper.DateParser
	repo        storage.Repository
	checksumGen *checksum.Generator
}

func NewOrchestrator(
	cfg *config.Config,
	logger *observability.Logger,
	f *fetcher.Fetcher,
	s *scraper.Scraper,
	dp *scraper.DateParser,
	repo storage.Repository,
	checksumGen *checksum.Generator,
) *Orchestrator {
	return &Orchestrator{
		cfg:         cfg,
		logger:      logger,
		fetcher:     f,
		scraper:     s,
		dateParser:  dp,
		repo:        repo,
		checksumGen: checksumGen,
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
func (o *Orchestrator) Run(ctx context.Context, langCfg *config.LanguageConfig) (*PaginationStats, error) {
	baseURL := langCfg.BaseURL
	maxPages := langCfg.MaxPages
	latestKnownDate := time.Now().UTC().AddDate(0, 0, -o.cfg.Pagination.DaysBackThreshold).Truncate(24 * time.Hour)

	o.logger.Info("Starting pagination",
		"language", langCfg.Name,
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
			"language", langCfg.Name,
			"page", pageNum,
			"url", currentURL,
		)

		// Фетчим страницу
		resp, err := o.fetcher.Fetch(ctx, currentURL, langCfg.Name)
		if err != nil {
			o.logger.Error("Fetch failed",
				"language", langCfg.Name,
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
				"language", langCfg.Name,
				"page", pageNum,
				"error", err.Error(),
			)
			stats.StoppedReason = fmt.Sprintf("parse error at page %d: %v", pageNum, err)
			return stats, err
		}

		if len(cards) == 0 {
			o.logger.Info("No cards found on page",
				"language", langCfg.Name,
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
					"language", langCfg.Name,
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
				"language", langCfg.Name,
				"page", pageNum,
				"card_num", i+1,
				"title", card.Title,
				"date", cardDate.Format("2006-01-02"),
				"url", card.URL,
				"thumbnail_url", card.ThumbnailURL,
			)

			// Если карточка новая — сохраняем в БД
			if !cardDate.Before(latestKnownDate) {
				articleCard := &storage.ArticleCard{
					CanonicalURL: card.URL,
					Title:        card.Title,
					Text:         card.Text,
					ImageURL:     card.ThumbnailURL,
					Date:         cardDate,
					Language:     langCfg.Name,
					SequenceNum:  card.SequenceNum,
					CheckSum:     o.checksumGen.GenerateContentHash(card.SequenceNum, cardDate.Format("2006-01-02"), card.Title, card.Text, []byte{}),
				}

				isNew, isUpdated, err := o.repo.UpsertCard(ctx, articleCard)
				if err != nil {
					o.logger.Error("Failed to upsert card",
						"language", langCfg.Name,
						"url", card.URL,
						"error", err.Error(),
					)
				} else {
					if isNew {
						o.logger.Debug("Card saved (new)", "url", card.URL)
					} else if isUpdated {
						o.logger.Debug("Card updated", "url", card.URL)
					}
				}
			}

			if cardDate.Before(latestKnownDate) {
				oldCardsOnPage++
			}
		}

		stats.OldCards += oldCardsOnPage

		o.logger.Info("Page analysis",
			"language", langCfg.Name,
			"page", pageNum,
			"total_cards", len(cards),
			"old_cards", oldCardsOnPage,
			"consecutive_old_pages", consecutiveOldPages,
		)

		// Проверяем, все ли карточки на странице старые
		if oldCardsOnPage == len(cards) {
			consecutiveOldPages++
			o.logger.Info("All cards on page are old",
				"language", langCfg.Name,
				"page", pageNum,
				"consecutive_old_pages", consecutiveOldPages,
			)

			if consecutiveOldPages >= o.cfg.Pagination.StopOnKnownChainPages {
				o.logger.Info("Stopping: reached consecutive old pages threshold",
					"language", langCfg.Name,
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
				"language", langCfg.Name,
				"page", pageNum,
			)
		}

		// Ищем ссылку на следующую страницу
		nextLink, err := o.scraper.FindNextPageLink(string(resp.Body))
		if err != nil {
			o.logger.Error("Failed to extract next link",
				"language", langCfg.Name,
				"page", pageNum,
				"error", err.Error(),
			)
			stats.StoppedReason = fmt.Sprintf("failed to extract next link at page %d: %v", pageNum, err)
			break
		}

		if nextLink == "" {
			o.logger.Info("No next link found",
				"language", langCfg.Name,
				"page", pageNum,
			)
			stats.StoppedReason = fmt.Sprintf("no next link at page %d", pageNum)
			break
		}

		currentURL = nextLink
		o.logger.Debug("Next URL extracted",
			"language", langCfg.Name,
			"page", pageNum,
			"next_url", nextLink,
		)
	}

	o.logger.Info("Pagination completed",
		"language", langCfg.Name,
		"total_pages", stats.TotalPages,
		"total_cards", stats.TotalCards,
		"old_cards", stats.OldCards,
		"reason", stats.StoppedReason,
	)

	return stats, nil
}
