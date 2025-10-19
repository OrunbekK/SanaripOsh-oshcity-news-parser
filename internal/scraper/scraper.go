package scraper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"oshcity-news-parser/internal/observability"
)

type Scraper struct {
	selectors *Selectors
	debugDir  string
	logger    *observability.Logger
}

func NewScraper(selectors *Selectors, debugDir string, logger *observability.Logger) *Scraper {
	return &Scraper{
		selectors: selectors,
		debugDir:  debugDir,
		logger:    logger,
	}
}

// ParseListing парсит листинг и возвращает массив карточек
func (s *Scraper) ParseListing(html string) ([]*Card, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var cards []*Card
	sequenceNum := 0

	doc.Find(s.selectors.CardSelectors).Each(func(i int, sel *goquery.Selection) {
		card := &Card{
			SequenceNum: sequenceNum,
		}

		s.logger.Debug("Processing card", "card_num", sequenceNum)

		// Title
		card.Title = trySelectors(sel, s.selectors.TitleSelectors)
		if card.Title == "" {
			html, _ := sel.Html()
			s.logger.Debug("Card skipped: no title")
			s.saveDebugCard(sequenceNum, "(no title)", html, "no_title")

			if card.DateRaw != "" {
				card.Title = card.DateRaw
			} else if card.Text != "" {
				// Берём до первой точки, или до восклицательного знака, или первые 500 символов
				text := strings.TrimSpace(card.Text)

				// Ищем первую точку
				if idx := strings.Index(text, "."); idx > 0 {
					card.Title = text[:idx]
				} else if idx := strings.Index(text, "!"); idx > 0 {
					card.Title = text[:idx]
				} else if len(text) > 500 {
					card.Title = text[:500] + "..."
				} else {
					card.Title = text
				}

			} else {
				return
			}
		}

		// URL
		urlRaw := trySelectors(sel, s.selectors.URLSelectors)
		if urlRaw == "" {
			s.logger.Debug("Card skipped: no url")
			s.saveDebugCard(sequenceNum, card.Title, html, "no_url")
			return
		}
		card.URL = normalizeURL(urlRaw)

		// ThumbnailURL
		thumbRaw := trySelectorsAttr(sel, s.selectors.ImageSelectors, "src")
		if thumbRaw == "" {
			s.logger.Debug("Card skipped: no thumb")
			s.saveDebugCard(sequenceNum, html, card.Title, "no_thumb")
			return
		}
		card.ThumbnailURL = normalizeURL(thumbRaw)

		// Text (превью из листинга)
		card.Text = trySelectors(sel, s.selectors.TextSelectors)
		if card.Text == "" {
			// Если нет text, используем title как текст
			if card.Title != "" {
				card.Text = card.Title
			} else {
				html, _ := sel.Html()
				s.logger.Debug("Card skipped: no text and no title")
				s.saveDebugCard(sequenceNum, "(no text/title)", html, "no_text_title")
				return
			}
		}

		// Date
		card.DateRaw = trySelectors(sel, s.selectors.DateSelectors)
		if card.DateRaw == "" {
			s.logger.Debug("Card skipped: no date")
			s.saveDebugCard(sequenceNum, html, card.Title, "no_date")
			return
		}

		cards = append(cards, card)
		sequenceNum++
	})

	s.logger.Debug("ParseListing completed", "total_cards", len(cards))

	return cards, nil
}

// FindNextPageLink ищет ссылку на следующую страницу
func (s *Scraper) FindNextPageLink(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	for _, selectorStr := range s.selectors.NextPageLink {
		// Парсим селектор с атрибутом (например "a.next@href")
		selector := selectorStr
		attr := "href" // по умолчанию

		if strings.Contains(selectorStr, "@") {
			parts := strings.Split(selectorStr, "@")
			if len(parts) == 2 {
				selector = parts[0]
				attr = parts[1]
			}
		}

		// Ищем элемент и берём атрибут
		href, exists := doc.Find(selector).First().Attr(attr)
		if exists && href != "" {
			s.logger.Debug("Found next page link",
				"selector", selectorStr,
				"href", href,
			)
			return normalizeURL(href), nil
		}

		s.logger.Debug("Next page link selector not found",
			"selector", selectorStr,
		)
	}

	return "", nil // Нет следующей страницы
}

// trySelectorsAttr пробует селекторы и возвращает атрибут (например @src)
func trySelectorsAttr(sel *goquery.Selection, selectors []string, attr string) string {
	for _, selector := range selectors {
		parts := strings.Split(selector, "@")
		if len(parts) == 2 {
			selector = parts[0]
			attr = parts[1]
		}

		value, exists := sel.Find(selector).First().Attr(attr)
		if exists && value != "" {
			// Если это srcset, парсим первый URL
			if attr == "srcset" {
				urls := strings.Split(value, ",")
				for _, urlPair := range urls {
					parts := strings.Fields(strings.TrimSpace(urlPair))
					if len(parts) > 0 {
						url := parts[0]
						if !strings.HasPrefix(url, "data:") { // Пропускаем data-URI
							return url
						}
					}
				}
			}
			return value
		}
	}
	return ""
}

func trySelectors(sel *goquery.Selection, selectors []string) string {
	for _, selector := range selectors {
		// Если селектор содержит @, парсим как атрибут
		if strings.Contains(selector, "@") {
			parts := strings.Split(selector, "@")
			if len(parts) == 2 {
				selector = parts[0]
				attr := parts[1]
				value, exists := sel.Find(selector).First().Attr(attr)
				if exists && value != "" {
					return value
				}
			}
			continue
		}

		text := strings.TrimSpace(sel.Find(selector).First().Text())
		if text != "" {
			return text
		}
	}
	return ""
}

func normalizeURL(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	// Удаляем якори и параметры для чистоты
	if idx := strings.Index(urlStr, "#"); idx > -1 {
		urlStr = urlStr[:idx]
	}
	return urlStr
}

func (s *Scraper) saveDebugCard(sequenceNum int, cardTitle string, html string, reason string) {
	debugDir := filepath.Dir(s.debugDir)
	debugDir = filepath.Join(debugDir, "debug")

	// Создаём директорию, если её нет
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		s.logger.Error("Failed to create debug directory", "dir", debugDir, "error", err.Error())
		return
	}
	// Генерируем уникальное имя файла
	filename := ""
	for i := 0; i < 100; i++ {
		candidateName := fmt.Sprintf("%s/card_%s_%d.html", debugDir, reason, i)
		if _, err := os.Stat(candidateName); os.IsNotExist(err) {
			filename = candidateName
			break
		}
	}

	if filename == "" {
		s.logger.Error("Failed to generate unique debug filename")
		return
	}

	// Сохраняем HTML
	if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
		s.logger.Error("Failed to write debug file", "error", err.Error())
		return
	}

	s.logger.Debug("Debug card saved",
		"card_num", sequenceNum,
		"card_title", cardTitle,
		"reason", reason,
		"file", filename,
	)
}
