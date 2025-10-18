package normalize

import (
	"strings"
	"testing"

	"oshcity-news-parser/internal/config"
)

func TestTruncatePreview(t *testing.T) {
	cfg := &config.Config{
		Normalize: config.NormalizeConfig{
			MaxPreviewChars: 50,
		},
	}

	normalizer := NewNormalizer(cfg)

	input := "Это очень длинный текст который должен быть обрезан по лимиту символов"
	result := normalizer.TruncatePreview(input)

	if len(result) > 50 {
		t.Errorf("TruncatePreview result too long: %d > 50", len(result))
	}

	if !strings.Contains(result, "…") {
		t.Errorf("TruncatePreview should end with …")
	}
}

func TestCleanHTML(t *testing.T) {
	cfg := &config.Config{
		Normalize: config.NormalizeConfig{
			TrimNBSP:       true,
			CollapseSpaces: true,
		},
	}

	normalizer := NewNormalizer(cfg)

	html := `
		<article>
			<p>Текст&nbsp;&nbsp;&nbsp;с&nbsp;NBSP</p>
			<script>alert('xss')</script>
			<p>Еще текст   с    пробелами</p>
		</article>
	`

	result := normalizer.cleanHTML(html)

	// Проверяем что нет script
	if strings.Contains(result, "script") || strings.Contains(result, "alert") {
		t.Errorf("Script tag not removed")
	}

	// Проверяем что NBSP заменены
	if strings.Contains(result, "\u00A0") {
		t.Errorf("NBSP not replaced")
	}

	// Проверяем что пробелы схлопаны
	if strings.Contains(result, "   ") {
		t.Errorf("Multiple spaces not collapsed")
	}
}
