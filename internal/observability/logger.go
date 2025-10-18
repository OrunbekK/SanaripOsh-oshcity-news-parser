package observability

import (
	"log"
)

type Logger struct {
	// Placeholder
}

func NewLogger(logPath, logLevel string) *Logger {
	return &Logger{}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	log.Println(msg, fields)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	log.Println("ERROR:", msg, fields)
}
