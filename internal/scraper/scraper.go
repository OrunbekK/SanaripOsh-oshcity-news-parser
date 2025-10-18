package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Scraper struct {
	selectors *Selectors
}

func NewScraper(selectors *Selectors) *Scraper {
	return &Scraper{
		selectors: selectors,
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

	// DEBUG: выводим что нашли по селектору
	fmt.Printf("DEBUG: Ищем по селектору: %s\n", s.selectors.CardSelectors)
	count := doc.Find(s.selectors.CardSelectors).Length()
	fmt.Printf("DEBUG: Найдено элементов: %d\n", count)

	// Найти контейнер со списком
	doc.Find(s.selectors.CardSelectors).Each(func(i int, sel *goquery.Selection) {
		card := &Card{
			SequenceNum: sequenceNum,
		}

		// Title: пробуем селекторы по очереди
		card.Title = trySelectors(sel, s.selectors.TitleSelectors)
		if card.Title == "" {
			return // Пропуск если нет title
		}

		// URL: пробуем селекторы
		urlRaw := trySelectors(sel, s.selectors.URLSelectors)
		if urlRaw == "" {
			return
		}
		// Нормализуем URL (убираем якоря, параметры)
		card.URL = normalizeURL(urlRaw)

		// Date: пробуем селекторы
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

func trySelectors(s *goquery.Selection, selectors []string) string {
	for _, selector := range selectors {
		text := strings.TrimSpace(s.Find(selector).First().Text())
		if text != "" {
			return text
		}
		// Пробуем атрибут (для ссылок)
		attr, exists := s.Find(selector).First().Attr("href")
		if exists && attr != "" {
			return attr
		}
		attr, exists = s.Find(selector).First().Attr("src")
		if exists && attr != "" {
			return attr
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
