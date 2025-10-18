package scraper

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// Русские месяцы
	ruMonths = map[string]int{
		"января":   1,
		"февраля":  2,
		"марта":    3,
		"апреля":   4,
		"мая":      5,
		"июня":     6,
		"июля":     7,
		"августа":  8,
		"сентября": 9,
		"октября":  10,
		"ноября":   11,
		"декабря":  12,
	}

	// Киргизские месяцы
	kyMonths = map[string]int{
		"январь":   1,
		"февраль":  2,
		"март":     3,
		"апрель":   4,
		"май":      5,
		"июнь":     6,
		"июль":     7,
		"август":   8,
		"сентябрь": 9,
		"октябрь":  10,
		"ноябрь":   11,
		"декабрь":  12,
	}

	// Названия дней недели (для удаления)
	ruDays    = []string{"пн", "вт", "ср", "чт", "пт", "сб", "вс", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота", "воскресенье"}
	kyDays    = []string{"дүй", "шейш", "шарш", "бейш", "жума", "ишемби", "чекмек"}
	ruToday   = []string{"сегодня", "сейчас"}
	kyToday   = []string{"бүгүн"}
	ruYesDays = []string{"вчера", "вчерашний"}
	kyYesDays = []string{"кечээ"}
)

type DateParser struct {
	lang string // "ru" или "ky"
}

func NewDateParser(lang string) *DateParser {
	return &DateParser{lang: lang}
}

// Parse парсит дату RU/KY и возвращает time.Time (UTC, время 00:00:00)
func (dp *DateParser) Parse(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Очистка от дней недели
	if dp.lang == "ru" {
		for _, day := range ruDays {
			dateStr = strings.ReplaceAll(strings.ToLower(dateStr), day, "")
		}
	} else {
		for _, day := range kyDays {
			dateStr = strings.ReplaceAll(strings.ToLower(dateStr), day, "")
		}
	}

	dateStr = strings.TrimSpace(dateStr)

	// Проверка "сегодня" / "бүгүн"
	lowerDate := strings.ToLower(dateStr)
	if dp.lang == "ru" {
		for _, today := range ruToday {
			if strings.Contains(lowerDate, today) {
				return time.Now().UTC().Truncate(24 * time.Hour), nil
			}
		}
		for _, yesterday := range ruYesDays {
			if strings.Contains(lowerDate, yesterday) {
				return time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour), nil
			}
		}
	} else {
		for _, today := range kyToday {
			if strings.Contains(lowerDate, today) {
				return time.Now().UTC().Truncate(24 * time.Hour), nil
			}
		}
		for _, yesterday := range kyYesDays {
			if strings.Contains(lowerDate, yesterday) {
				return time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour), nil
			}
		}
	}

	// Форматы для русского
	if dp.lang == "ru" {
		return dp.parseRussian(dateStr)
	}

	// Форматы для киргизского
	return dp.parseKyrgyz(dateStr)
}

func (dp *DateParser) parseRussian(dateStr string) (time.Time, error) {
	dateStr = strings.ToLower(dateStr)

	// Формат: "18 октября 2024" или "18 октября"
	re := regexp.MustCompile(`(\d{1,2})\s+([а-яё]+)\s+(\d{4})?`)
	matches := re.FindStringSubmatch(dateStr)
	if len(matches) > 0 {
		day, err := parseIntSafe(matches[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid day: %q: %w", matches[1], err)
		}

		monthName := matches[2]
		month, ok := ruMonths[monthName]
		if !ok {
			return time.Time{}, fmt.Errorf("unknown month: %s", monthName)
		}

		year := time.Now().Year()
		if yearStr := matches[3]; yearStr != "" {
			y, err := parseIntSafe(yearStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid year: %q: %w", yearStr, err)
			}
			year = y
		}

		// Опционально: проверка валидности даты
		if day < 1 || day > 31 {
			return time.Time{}, fmt.Errorf("invalid day: %d", day)
		}
		if month < 1 || month > 12 {
			return time.Time{}, fmt.Errorf("invalid month: %d", month)
		}

		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return t, nil
	}

	// Формат: "18.10.2024" или "18.10"
	re = regexp.MustCompile(`(\d{1,2})\.(\d{1,2})\.(\d{4})?`)
	matches = re.FindStringSubmatch(dateStr)
	if len(matches) > 0 {
		day, err := parseIntSafe(matches[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid day: %q: %w", matches[1], err)
		}

		monthNum, err := parseIntSafe(matches[2])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid month: %q: %w", matches[2], err)
		}

		year := time.Now().Year()
		if yearStr := matches[3]; yearStr != "" {
			y, err := parseIntSafe(yearStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid year: %q: %w", yearStr, err)
			}
			year = y
		}

		if day < 1 || day > 31 {
			return time.Time{}, fmt.Errorf("invalid day: %d", day)
		}
		if monthNum < 1 || monthNum > 12 {
			return time.Time{}, fmt.Errorf("invalid month: %d", monthNum)
		}

		t := time.Date(year, time.Month(monthNum), day, 0, 0, 0, 0, time.UTC)
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date (RU): %s", dateStr)
}

func (dp *DateParser) parseKyrgyz(dateStr string) (time.Time, error) {
	dateStr = strings.ToLower(dateStr)

	// Формат: "18 октябрь 2024" или "18 октябрь"
	re := regexp.MustCompile(`(\d{1,2})\s+([а-яё]+)\s+(\d{4})?`)
	matches := re.FindStringSubmatch(dateStr)
	if len(matches) > 0 {
		day, err := parseIntSafe(matches[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid day: %q: %w", matches[1], err)
		}

		monthName := matches[2]
		yearStr := matches[3]

		month, ok := kyMonths[monthName]
		if !ok {
			return time.Time{}, fmt.Errorf("unknown month (KY): %s", monthName)
		}

		year := time.Now().Year()
		if yearStr != "" {
			y, err := parseIntSafe(yearStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid year: %q: %w", yearStr, err)
			}
			year = y
		}

		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return t, nil
	}

	// Формат: "18.10.2024"
	re = regexp.MustCompile(`(\d{1,2})\.(\d{1,2})\.(\d{4})?`)
	matches = re.FindStringSubmatch(dateStr)
	if len(matches) > 0 {
		day := parseIntSafe(matches[1])
		month := parseIntSafe(matches[2])
		yearStr := matches[3]

		year := time.Now().Year()
		if yearStr != "" {
			year = parseIntSafe(yearStr)
		}

		t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date (KY): %s", dateStr)
}

func parseIntSafe(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %q as int: %w", s, err)
	}
	return result, nil
}
