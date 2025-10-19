package checksum

import (
	"crypto/sha256"
	"fmt"
	"unicode/utf16"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateContentHash генерирует контрольную сумму по формату OshCity
// TODO: ВАЖНО - Формат совместимости с C# модулем (OshSanaripWebSiteProcessor)
// Checksum = SHA256(sequenceNum) + SHA256(date_iso) + SHA256(title) + SHA256(text) + SHA256(imageBytes)
// Каждый хеш конкатенируется (5 хешей x 64 символа = 320 символов total)
func (g *Generator) GenerateContentHash(sequenceNum int, dateISO, title, text string, imageBytes []byte) string {
	hash1 := g.sha256String(fmt.Sprintf("%d", sequenceNum))
	hash2 := g.sha256String(dateISO)
	hash3 := g.sha256String(title)
	hash4 := g.sha256String(text)
	hash5 := g.sha256Bytes(imageBytes)

	return hash1 + hash2 + hash3 + hash4 + hash5
}

func (g *Generator) sha256String(input string) string {
	// Кодируем в UTF-16 LE как в C#
	runes := []rune(input)
	utf16Bytes := utf16.Encode(runes)

	// Конвертируем в byte array (little-endian)
	byteArray := make([]byte, len(utf16Bytes)*2)
	for i, r := range utf16Bytes {
		byteArray[i*2] = byte(r)
		byteArray[i*2+1] = byte(r >> 8)
	}

	hash := sha256.Sum256(byteArray)
	return fmt.Sprintf("%X", hash)
}

func (g *Generator) sha256Bytes(input []byte) string {
	hash := sha256.Sum256(input)
	return fmt.Sprintf("%X", hash)
}

// VerifyContentHash проверяет соответствие контрольной суммы
// TODO: ВАЖНО - Формат совместимости с C# модулем (OshSanaripWebSiteProcessor)
// Checksum = SHA256(sequenceNum) + SHA256(date_iso) + SHA256(title) + SHA256(text) + SHA256(imageBytes)
// Каждый хеш конкатенируется (5 хешей x 64 символа = 320 символов total)
func (g *Generator) VerifyContentHash(expectedHash string, sequenceNum int, dateISO, title, text string, imageBytes []byte) bool {
	computed := g.GenerateContentHash(sequenceNum, dateISO, title, text, imageBytes)
	return computed == expectedHash
}
