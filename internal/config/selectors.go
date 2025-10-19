package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"oshcity-news-parser/internal/scraper"
)

// LoadSelectors загружает селекторы из YAML файла
func LoadSelectors(filePath string) (*scraper.Selectors, error) {
	if filePath == "" {
		return nil, fmt.Errorf("selectors file path is empty")
	}

	// Проверяем существование файла
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("selectors file not found: %s: %w", filePath, err)
	}

	// Открываем файл
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open selectors file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close selectors file: %v\n", closeErr)
		}
	}()

	// Парсим YAML
	var selectors scraper.Selectors
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&selectors); err != nil {
		return nil, fmt.Errorf("failed to parse selectors YAML: %w", err)
	}

	// Валидируем селекторы
	if err := validateSelectors(&selectors); err != nil {
		return nil, err
	}

	return &selectors, nil
}

// LoadSelectorsForLanguage загружает селекторы на основе конфига и языка
func (c *Config) LoadSelectorsForLanguage(lang string) (*scraper.Selectors, error) {
	var filePath string

	if lang == "ky" {
		filePath = c.SelectorsFile.KY
	} else if lang == "ru" {
		filePath = c.SelectorsFile.RU
	} else {
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	// Если путь относительный, делаем его относительно конфига
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join("configs", filePath)
	}

	return LoadSelectors(filePath)
}

// validateSelectors проверяет минимальный набор селекторов
func validateSelectors(s *scraper.Selectors) error {
	if s.CardSelectors == "" {
		return fmt.Errorf("card_selectors is required")
	}
	if len(s.TitleSelectors) == 0 {
		return fmt.Errorf("title_selectors is required")
	}
	if len(s.URLSelectors) == 0 {
		return fmt.Errorf("url_selectors is required")
	}
	if len(s.ImageSelectors) == 0 {
		return fmt.Errorf("image_selectors is required")
	}
	if len(s.TextSelectors) == 0 {
		return fmt.Errorf("text_selectors is required")
	}
	if len(s.DateSelectors) == 0 {
		return fmt.Errorf("date_selectors is required")
	}
	if len(s.NextPageLink) == 0 {
		return fmt.Errorf("next_page_link is required")
	}

	return nil
}
