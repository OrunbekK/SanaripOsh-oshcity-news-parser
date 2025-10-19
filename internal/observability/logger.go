package observability

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(logPath, logLevel string, maxAgeDays, maxSizeMB, maxBackups int) *Logger {
	// Создаём директорию логов если её нет
	if logPath != "" {
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			slog.Error("Failed to create log directory", "error", err)
		}
	}

	var output io.Writer = os.Stdout

	// Если указан путь логов — добавляем ротацию
	if logPath != "" {
		writer := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    maxSizeMB,
			MaxAge:     maxAgeDays,
			MaxBackups: maxBackups,
			Compress:   true,
			LocalTime:  true,
		}
		// Выводим одновременно в файл и консоль
		output = io.MultiWriter(os.Stdout, writer)
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: parseLevel(logLevel),
	})

	return &Logger{
		logger: slog.New(handler),
	}
}

func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
