package checksum

import (
	"testing"
	"time"
)

func TestGenerateContentHash(t *testing.T) {
	gen := NewGenerator()

	url := "https://example.com/news/123"
	title := "Тестовая новость"
	text := "Содержание новости"
	date := time.Date(2025, 10, 18, 0, 0, 0, 0, time.UTC)

	hash1 := gen.GenerateContentHash(url, title, text, date)
	hash2 := gen.GenerateContentHash(url, title, text, date)

	// Хеш должен быть детерминированным
	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %s != %s", hash1, hash2)
	}

	// Хеш должен быть 64 символа (SHA256 hex)
	if len(hash1) != 64 {
		t.Errorf("Hash wrong length: %d, expected 64", len(hash1))
	}

	// Изменение контента должно изменить хеш
	hash3 := gen.GenerateContentHash(url, "Другой заголовок", text, date)
	if hash1 == hash3 {
		t.Errorf("Hash should change when title changes")
	}
}

func TestVerifyContentHash(t *testing.T) {
	gen := NewGenerator()

	url := "https://example.com/news/123"
	title := "Тестовая новость"
	text := "Содержание новости"
	date := time.Date(2025, 10, 18, 0, 0, 0, 0, time.UTC)

	hash := gen.GenerateContentHash(url, title, text, date)

	// Проверка с правильными данными
	if !gen.VerifyContentHash(hash, url, title, text, date) {
		t.Errorf("VerifyContentHash failed for correct data")
	}

	// Проверка с неправильным заголовком
	if gen.VerifyContentHash(hash, url, "Другой заголовок", text, date) {
		t.Errorf("VerifyContentHash should fail for wrong title")
	}
}
