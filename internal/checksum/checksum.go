package checksum

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateContentHash генерирует SHA256 хеш контента
// Формула: SHA256(url|title|text|date_iso)
func (g *Generator) GenerateContentHash(url, title, text string, date time.Time) string {
	// Нормализуем дату в ISO формат (без времени)
	dateISO := date.UTC().Format("2006-01-02")

	// Конкатенируем: url|title|text|date
	content := fmt.Sprintf("%s|%s|%s|%s", url, title, text, dateISO)

	// Вычисляем SHA256
	hash := sha256.Sum256([]byte(content))

	// Возвращаем hex
	return fmt.Sprintf("%x", hash)
}

// VerifyContentHash проверяет соответствие хеша
func (g *Generator) VerifyContentHash(expectedHash, url, title, text string, date time.Time) bool {
	computed := g.GenerateContentHash(url, title, text, date)
	return computed == expectedHash
}
