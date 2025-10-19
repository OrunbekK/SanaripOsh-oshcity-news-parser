package main

import (
	"context"
	"errors"
	"log"
	"os"
	"oshcity-news-parser/internal/checksum"
	"oshcity-news-parser/internal/storage"
	"oshcity-news-parser/internal/storage/mssql"
	"time"

	"oshcity-news-parser/internal/app"
	"oshcity-news-parser/internal/config"
	"oshcity-news-parser/internal/fetcher"
	"oshcity-news-parser/internal/observability"
	"oshcity-news-parser/internal/scraper"
)

func main() {
	configPath := "configs/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Загружаем конфиг
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Инициализируем logger
	logger := observability.NewLogger(
		cfg.Observability.LogPath,
		cfg.Observability.LogLevel,
		cfg.Observability.MaxLogAgeDays,
		cfg.Observability.MaxLogSizeMB,
		cfg.Observability.MaxBackups,
	)
	logger.Info("Application started", "config", configPath)

	// Инициализируем fetcher
	f := fetcher.NewFetcher(cfg, logger)
	defer func() {
		logger.Info("Closing fetcher")
		if err := f.Close(); err != nil {
			logger.Error("Failed to close fetcher", "error", err.Error())
		}
	}()

	// Инициализируем Repository
	var repo storage.Repository
	if cfg.Storage.Driver == "mssql" {
		mssqlRepo, err := mssql.NewRepository(
			cfg.Storage.DSN,
			cfg.Storage.CommandTimeoutMS,
			cfg.Storage.MaxOpenConnections,
			cfg.Storage.MaxIdleConnections,
			cfg.Storage.ConnectionMaxLifetime,
			cfg.Storage.ConnectionMaxIdleTime,
			logger,
		)
		if err != nil {
			log.Fatalf("Failed to initialize MS SQL repository: %v", err)
		}
		defer func() {
			logger.Info("Closing repository")
			if err := mssqlRepo.Close(); err != nil {
				logger.Error("Failed to close repository", "error", err.Error())
			}
		}()
		repo = mssqlRepo
	} else {
		log.Fatalf("Unsupported storage driver: %s", cfg.Storage.Driver)
	}

	// Инициализируем generator checksum
	checksumGen := checksum.NewGenerator()

	// Настраиваем graceful shutdown с таймаутом 60 секунд
	shutdownTimeout := time.Duration(cfg.Scheduler.GracefulShutdownTimeoutS) * time.Second
	ctx, cancel := app.GracefulShutdown(logger, shutdownTimeout)
	defer cancel()

	logger.Info("Starting pagination", "languages_count", len(cfg.Languages))

	// Запускаем пагинацию для каждого языка из конфига
	for _, langCfg := range cfg.Languages {
		// Проверяем был ли сигнал shutdown
		select {
		case <-ctx.Done():
			logger.Info("Shutdown signal detected, stopping pagination")
			break
		default:
		}

		logger.Info("Processing language", "language", langCfg.Name)

		// Загружаем селекторы
		selectors, err := cfg.LoadSelectorsForLanguage(&langCfg)
		if err != nil {
			logger.Error("Failed to load selectors", "language", langCfg.Name, "error", err.Error())
			continue
		}

		// Создаём компоненты для языка
		scr := scraper.NewScraper(selectors, logger)
		dateParser := scraper.NewDateParser(langCfg.Name)
		orchestrator := app.NewOrchestrator(cfg, logger, f, scr, dateParser, repo, checksumGen)

		// Запускаем пагинацию с передачей context
		stats, err := orchestrator.Run(ctx, &langCfg)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("Pagination cancelled by shutdown signal", "language", langCfg.Name)
			} else {
				logger.Error("Pagination failed", "language", langCfg.Name, "error", err.Error())
			}
		} else {
			logger.Info("Pagination completed",
				"language", langCfg.Name,
				"total_pages", stats.TotalPages,
				"total_cards", stats.TotalCards,
				"old_cards", stats.OldCards,
				"reason", stats.StoppedReason,
			)
		}
	}

	// Обновляем контрольные суммы новостей в БД
	logger.Info("Updating news checksums in database")
	msg, err := repo.UpdateNewsCheckSum(ctx)
	if err != nil {
		logger.Error("Failed to update news checksums", "error", err.Error())
	} else {
		logger.Info("News checksums updated successfully", "message", msg)
	}

	logger.Info("Application finished")
}
