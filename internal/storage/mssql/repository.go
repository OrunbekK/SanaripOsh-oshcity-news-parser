package mssql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/microsoft/go-mssqldb"

	"oshcity-news-parser/internal/observability"
	"oshcity-news-parser/internal/storage"
)

type Repository struct {
	db             *sql.DB
	commandTimeout time.Duration
	logger         *observability.Logger
}

func NewRepository(dsn string, commandTimeoutMS int, logger *observability.Logger) (*Repository, error) {
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Тестируем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{
		db:             db,
		commandTimeout: time.Duration(commandTimeoutMS) * time.Millisecond,
		logger:         logger,
	}, nil
}

// UpsertCard сохраняет или обновляет карточку
func (r *Repository) UpsertCard(ctx context.Context, card *storage.ArticleCard) (isNew bool, isUpdated bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, r.commandTimeout)
	defer cancel()

	// MERGE statement для MS SQL
	query := `
		MERGE INTO TblNews AS target
		USING (SELECT @URL AS URL) AS source
		ON target.[URL] = source.URL
		WHEN MATCHED THEN
			UPDATE SET
				[Title] = @Title,
				[Text] = @Text,
				[ThumbnailURL] = @ThumbnailURL,
				[DT] = @DT,
				[CheckSum] = @CheckSum,
				[SequenceNum] = @SequenceNum
		WHEN NOT MATCHED THEN
			INSERT ([Language_UID], [SequenceNum], [DT], [Title], [Text], [URL], [ThumbnailURL], [CheckSum])
			VALUES (@LanguageUID, @SequenceNum, @DT, @Title, @Text, @URL, @ThumbnailURL, @CheckSum);
	`

	// Получаем Language_UID по коду языка
	languageUID, err := r.getLanguageUID(ctx, card.Language)
	if err != nil {
		r.logger.Error("Failed to get language UID",
			"language", card.Language,
			"error", err.Error(),
		)
		return false, false, err
	}

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return false, false, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.logger.Error("Failed to close statement", "error", err.Error())
		}
	}()

	result, err := stmt.ExecContext(ctx,
		sql.Named("LanguageUID", languageUID),
		sql.Named("SequenceNum", card.SequenceNum),
		sql.Named("Title", card.Title),
		sql.Named("Text", card.Text),
		sql.Named("URL", card.CanonicalURL),
		sql.Named("ThumbnailURL", card.ImageURL),
		sql.Named("DT", card.Date),
		sql.Named("CheckSum", card.CheckSum),
	)

	if err != nil {
		return false, false, fmt.Errorf("failed to execute upsert: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Если вставлена новая строка
	if rowsAffected > 0 {
		isNew = true
	} else {
		// Если обновлена существующая
		isUpdated = true
	}

	return isNew, isUpdated, nil
}

// ExistsByURL проверяет наличие карточки по URL
func (r *Repository) ExistsByURL(ctx context.Context, url string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.commandTimeout)
	defer cancel()

	query := `SELECT COUNT(*) FROM TblNews WHERE URL = @URL`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.logger.Error("Failed to close statement", "error", err.Error())
		}
	}()

	var count int
	err = stmt.QueryRowContext(ctx, sql.Named("URL", url)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query database: %w", err)
	}

	return count > 0, nil
}

// GetLatestKnownDate получает последнюю загруженную дату для языка
func (r *Repository) GetLatestKnownDate(ctx context.Context, lang string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, r.commandTimeout)
	defer cancel()

	languageUID, err := r.getLanguageUID(ctx, lang)
	if err != nil {
		return time.Time{}, err
	}

	query := `SELECT MAX(DT) FROM TblNews WHERE Language_UID = @LanguageUID`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.logger.Error("Failed to close statement", "error", err.Error())
		}
	}()

	var latestDate sql.NullTime
	err = stmt.QueryRowContext(ctx, sql.Named("LanguageUID", languageUID)).Scan(&latestDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query database: %w", err)
	}

	if !latestDate.Valid {
		// Если нет данных, возвращаем время 1 года назад
		return time.Now().UTC().AddDate(-1, 0, 0), nil
	}

	return latestDate.Time, nil
}

// GetCardCount получает количество загруженных карточек для языка
func (r *Repository) GetCardCount(ctx context.Context, lang string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.commandTimeout)
	defer cancel()

	languageUID, err := r.getLanguageUID(ctx, lang)
	if err != nil {
		return 0, err
	}

	query := `SELECT COUNT(*) FROM TblNews WHERE Language_UID = @LanguageUID`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.logger.Error("Failed to close statement", "error", err.Error())
		}
	}()

	var count int
	err = stmt.QueryRowContext(ctx, sql.Named("LanguageUID", languageUID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to query database: %w", err)
	}

	return count, nil
}

// getLanguageUID получает UID языка по коду (ru, ky)
func (r *Repository) getLanguageUID(ctx context.Context, langAlias string) (int, error) {
	query := `SELECT UID FROM TblRefLanguages WHERE Alias = @Alias`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.logger.Error("Failed to close statement", "error", err.Error())
		}
	}()

	var uid int
	err = stmt.QueryRowContext(ctx, sql.Named("Alias", langAlias)).Scan(&uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("language not found: %s", langAlias)
		}
		return 0, fmt.Errorf("failed to query database: %w", err)
	}

	return uid, nil
}

// Close закрывает соединение с БД
func (r *Repository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
