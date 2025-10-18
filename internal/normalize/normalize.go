package normalize

import (
	"oshcity-news-parser/internal/config"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Normalizer struct {
	cfg *config.Config
}

func NewNormalizer(cfg *config.Config) *Normalizer {
	return &Normalizer{cfg: cfg}
}

type ArticleContent struct {
	Title    string
	Text     string
	ImageURL string
	DateRaw  string
}

// ParseDetailPage парсит детальную страницу и извлекает контент
func (n *Normalizer) ParseDetailPage(html string) (*ArticleContent, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	content := &ArticleContent{}

	// Title: og:title или h1
	ogTitle, _ := doc.Find("meta[property='og:title']").Attr("content")
	if ogTitle != "" {
		content.Title = ogTitle
	} else {
		content.Title = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	// Image: og:image → первое img в контенте
	ogImage, _ := doc.Find("meta[property='og:image']").Attr("content")
	if ogImage != "" {
		content.ImageURL = ogImage
	} else {
		// Поиск первого img в основном контенте
		content.ImageURL, _ = doc.Find("article img, .post-content img, .entry-content img").First().Attr("src")
	}

	// Text: основное тело (article, .post-content, .entry-content)
	var textHTML string
	article := doc.Find("article").First()
	if article.Length() > 0 {
		textHTML, _ = article.Html()
	} else {
		postContent := doc.Find(".post-content, .entry-content, .content").First()
		if postContent.Length() > 0 {
			textHTML, _ = postContent.Html()
		} else {
			textHTML, _ = doc.Find("main").Html()
		}
	}

	// Удаляем мусорные блоки
	textHTML = n.stripBlocks(textHTML)

	// Очистка HTML → текст
	content.Text = n.cleanHTML(textHTML)

	return content, nil
}

// stripBlocks удаляет блоки типа "Похожие", "Видео" и т.д.
func (n *Normalizer) stripBlocks(html string) string {
	result := html

	// Удаляем блоки по содержимому
	for _, blockName := range n.cfg.Normalize.StripBlocks {
		// Ищем div/section/aside с классом или текстом содержащим blockName
		patterns := []string{
			`<div[^>]*>(\s*<h\d[^>]*>` + blockName + `</h\d>|` + blockName + `)[^<]*(?:<[^>]*>)*?</div>`,
			`<section[^>]*>(\s*<h\d[^>]*>` + blockName + `</h\d>|` + blockName + `)[^<]*(?:<[^>]*>)*?</section>`,
			`<aside[^>]*>[^<]*` + blockName + `[^<]*(?:<[^>]*>)*?</aside>`,
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(`(?i)` + pattern)
			result = re.ReplaceAllString(result, "")
		}
	}

	return result
}

// cleanHTML парсит HTML и извлекает текст
func (n *Normalizer) cleanHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}

	// Удаляем script, style, nav
	doc.Find("script, style, nav, footer, .ads, [class*='advertisement']").Remove()

	// Извлекаем текст
	text := doc.Text()

	if n.cfg.Normalize.TrimNBSP {
		// Заменяем NBSP (\u00A0) на обычный пробел
		text = strings.ReplaceAll(text, "\u00A0", " ")
	}

	if n.cfg.Normalize.CollapseSpaces {
		// Схлопываем множественные пробелы
		re := regexp.MustCompile(`\s+`)
		text = re.ReplaceAllString(text, " ")
	}

	text = strings.TrimSpace(text)

	return text
}

// TruncatePreview обрезает текст до maxPreviewChars
func (n *Normalizer) TruncatePreview(text string) string {
	if len(text) <= n.cfg.Normalize.MaxPreviewChars {
		return text
	}

	// Находим последний пробел перед лимитом
	truncated := text[:n.cfg.Normalize.MaxPreviewChars]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		return text[:lastSpace] + "…"
	}

	return truncated + "…"
}

// NormalizeURL нормализует URL (убирает якори, параметры если нужно)
func NormalizeURL(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	// Удаляем якори
	if idx := strings.Index(urlStr, "#"); idx > -1 {
		urlStr = urlStr[:idx]
	}
	return urlStr
}
