package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oshcity-news-parser/internal/observability"
)

// GracefulShutdown запускает мониторинг OS сигналов и возвращает context для отмены
func GracefulShutdown(logger *observability.Logger, shutdownTimeout time.Duration) (context.Context, context.CancelFunc) {
	// Создаём context с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Shutdown signal received", "signal", sig.String())
		cancel() // Отменяем context при получении сигнала
	}()

	return ctx, cancel
}

/*
// WaitForShutdown блокирует до получения сигнала завершения
func WaitForShutdown(logger *observability.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("Shutdown signal received", "signal", sig.String())
}
*/
