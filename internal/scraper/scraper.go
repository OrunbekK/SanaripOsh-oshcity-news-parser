package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"oshcity-news-parser/internal/observability"
)

type Scraper struct {
	selectors *Selectors
	logger    *observability.Logger
}

func NewScraper(selectors *Selectors, logger *observability.Logger) *Scraper {
	return &Scraper{
		selectors: selectors,
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

		// Title
		card.Title = trySelectors(sel, s.selectors.TitleSelectors)
		if card.Title == "" {
			return
		}

		// URL
		urlRaw := trySelectors(sel, s.selectors.URLSelectors)
		if urlRaw == "" {
			return
		}
		card.URL = normalizeURL(urlRaw)

		// ThumbnailURL
		thumbRaw := trySelectorsAttr(sel, s.selectors.ImageSelectors, "src")
		if thumbRaw == "" {
			return
		}
		card.ThumbnailURL = normalizeURL(thumbRaw)

		// Text (превью из листинга)
		card.Text = trySelectors(sel, s.selectors.TextSelectors)
		if card.Text == "" {
			return
		}

		// Date
		card.DateRaw = trySelectors(sel, s.selectors.DateSelectors)
		if card.DateRaw == "" {
			return
		}

		cards = append(cards, card)
		sequenceNum++
	})

	return cards, nil
}

// FindNextPageLink ищет ссылку на следующую страницу
func (s *Scraper) FindNextPageLink(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	for _, selector := range s.selectors.NextPageLink {
		href, exists := doc.Find(selector).Attr("href")
		if exists && href != "" {
			return normalizeURL(href), nil
		}
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
			fmt.Println(attr, value)
			if attr == "srcset" {
				urls := strings.Split(value, ",")
				fmt.Println(value)
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
