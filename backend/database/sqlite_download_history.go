package database

import (
	"aura/logging"
	"aura/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// DownloadHistoryEntry is one poster-set-per-run download history record.
type DownloadHistoryEntry struct {
	ID              int64                        `json:"id"`
	TMDB_ID         string                       `json:"tmdb_id"`
	LibraryTitle    string                       `json:"library_title"`
	Edition         string                       `json:"edition"`
	MediaItemTitle  string                       `json:"media_item_title"`
	MediaItemYear   int                          `json:"media_item_year"`
	SetID           string                       `json:"set_id"`
	SetTitle        string                       `json:"set_title"`
	SetCreator      string                       `json:"set_creator"`
	SelectedTypes   models.SelectedTypes         `json:"selected_types"`
	WarningCount    int                          `json:"warning_count"`
	ImagesSucceeded int                          `json:"images_succeeded"`
	ImagesFailed    int                          `json:"images_failed"`
	Status          string                       `json:"status"`
	FailedImages    []models.ImageDownloadResult `json:"failed_images"`
	CreatedAt       time.Time                    `json:"created_at"`
	RatingKey       string                       `json:"rating_key,omitempty"`
}

func (s *SQliteDB) InsertDownloadHistoryEntry(ctx context.Context, entry DownloadHistoryEntry) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx,
		fmt.Sprintf("Insert Download History Entry for '%s' (Set: %s)", entry.MediaItemTitle, entry.SetTitle),
		logging.LevelDebug)
	defer logAction.Complete()

	if s == nil || s.conn == nil {
		logAction.SetError("DB: connection is nil", "", map[string]any{})
		return *logAction.Error
	}

	if entry.FailedImages == nil {
		entry.FailedImages = []models.ImageDownloadResult{}
	}
	failedImagesJSON, err := json.Marshal(entry.FailedImages)
	if err != nil {
		logAction.SetError("Failed to marshal Download History failed images", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	selectedTypesJSON, err := json.Marshal(entry.SelectedTypes)
	if err != nil {
		logAction.SetError("Failed to marshal Download History selected types", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	status := entry.Status
	if status == "" {
		switch {
		case entry.ImagesFailed == 0:
			status = "success"
		case entry.ImagesSucceeded == 0:
			status = "error"
		default:
			status = "warning"
		}
	}

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		logAction.SetError("DB: TX BEGIN failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	defer func() { _ = tx.Rollback() }()

	// Same media item + edition + set overwrites its existing entry instead of piling up duplicates.
	var existingID int64
	findErr := tx.QueryRowContext(ctx, `
SELECT id FROM DownloadHistory
WHERE tmdb_id = ? AND library_title = ? AND edition = ? AND set_id = ?;
`, entry.TMDB_ID, entry.LibraryTitle, entry.Edition, entry.SetID).Scan(&existingID)

	if findErr != nil && findErr != sql.ErrNoRows {
		logAction.SetError("DB: SELECT existing DownloadHistory entry failed", findErr.Error(), map[string]any{"error": findErr.Error()})
		return *logAction.Error
	}

	if findErr == nil {
		_, err = tx.ExecContext(ctx, `
UPDATE DownloadHistory
SET media_item_title = ?, media_item_year = ?, set_title = ?, set_creator = ?, selected_types = ?, warning_count = ?,
    images_succeeded = ?, images_failed = ?, status = ?, failed_images = ?, created_at = CURRENT_TIMESTAMP
WHERE id = ?;
`,
			entry.MediaItemTitle, entry.MediaItemYear, entry.SetTitle, entry.SetCreator, string(selectedTypesJSON), entry.WarningCount,
			entry.ImagesSucceeded, entry.ImagesFailed, status, string(failedImagesJSON), existingID,
		)
		if err != nil {
			logAction.SetError("DB: UPDATE DownloadHistory failed", err.Error(), map[string]any{"error": err.Error()})
			return *logAction.Error
		}
	} else {
		_, err = tx.ExecContext(ctx, `
INSERT INTO DownloadHistory (
  tmdb_id, library_title, edition, media_item_title, media_item_year,
  set_id, set_title, set_creator, selected_types, warning_count,
  images_succeeded, images_failed, status, failed_images
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`,
			entry.TMDB_ID, entry.LibraryTitle, entry.Edition, entry.MediaItemTitle, entry.MediaItemYear,
			entry.SetID, entry.SetTitle, entry.SetCreator, string(selectedTypesJSON), entry.WarningCount,
			entry.ImagesSucceeded, entry.ImagesFailed, status, string(failedImagesJSON),
		)
		if err != nil {
			logAction.SetError("DB: INSERT DownloadHistory failed", err.Error(), map[string]any{"error": err.Error()})
			return *logAction.Error
		}
	}

	if err := tx.Commit(); err != nil {
		logAction.SetError("DB: TX COMMIT failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	return logging.LogErrorInfo{}
}

func (s *SQliteDB) GetDownloadHistory(ctx context.Context, limit, offset int) (entries []DownloadHistoryEntry, total int, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Get Download History", logging.LevelDebug)
	defer logAction.Complete()

	entries = []DownloadHistoryEntry{}

	if s == nil || s.conn == nil {
		logAction.SetError("DB: connection is nil", "", map[string]any{})
		return entries, 0, *logAction.Error
	}

	if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM DownloadHistory;`).Scan(&total); err != nil {
		logAction.SetError("DB: COUNT DownloadHistory failed", err.Error(), map[string]any{"error": err.Error()})
		return entries, 0, *logAction.Error
	}

	q := `SELECT ` + downloadHistoryColumns + `
FROM DownloadHistory
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
`
	rows, err := s.conn.QueryContext(ctx, q, limit, offset)
	if err != nil {
		logAction.SetError("DB: SELECT DownloadHistory failed", err.Error(), map[string]any{"error": err.Error()})
		return entries, 0, *logAction.Error
	}
	defer rows.Close()

	for rows.Next() {
		entry, err := scanDownloadHistoryEntry(rows)
		if err != nil {
			logAction.SetError("DB: scan DownloadHistory failed", err.Error(), map[string]any{"error": err.Error()})
			return entries, 0, *logAction.Error
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		logAction.SetError("DB: iterate DownloadHistory rows failed", err.Error(), map[string]any{"error": err.Error()})
		return entries, 0, *logAction.Error
	}

	logAction.AppendResult("count", len(entries))
	return entries, total, logging.LogErrorInfo{}
}

// GetDownloadHistoryEntry returns one Download History entry by ID
func (s *SQliteDB) GetDownloadHistoryEntry(ctx context.Context, id int64) (entry DownloadHistoryEntry, found bool, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf("Get Download History Entry %d", id), logging.LevelDebug)
	defer logAction.Complete()

	if s == nil || s.conn == nil {
		logAction.SetError("DB: connection is nil", "", map[string]any{})
		return entry, false, *logAction.Error
	}

	row := s.conn.QueryRowContext(ctx, `SELECT `+downloadHistoryColumns+` FROM DownloadHistory WHERE id = ?;`, id)
	entry, err := scanDownloadHistoryEntry(row)
	if err == sql.ErrNoRows {
		return entry, false, logging.LogErrorInfo{}
	}
	if err != nil {
		logAction.SetError("DB: SELECT DownloadHistory entry failed", err.Error(), map[string]any{"error": err.Error()})
		return entry, false, *logAction.Error
	}
	return entry, true, logging.LogErrorInfo{}
}

const downloadHistoryColumns = `id, tmdb_id, library_title, edition, media_item_title, media_item_year,
       set_id, set_title, set_creator, selected_types, warning_count,
       images_succeeded, images_failed, status, failed_images, created_at`

// scanDownloadHistoryEntry reads one row of downloadHistoryColumns
func scanDownloadHistoryEntry(row interface{ Scan(dest ...any) error }) (entry DownloadHistoryEntry, err error) {
	var failedImagesJSON, selectedTypesJSON string
	if err := row.Scan(
		&entry.ID, &entry.TMDB_ID, &entry.LibraryTitle, &entry.Edition, &entry.MediaItemTitle, &entry.MediaItemYear,
		&entry.SetID, &entry.SetTitle, &entry.SetCreator, &selectedTypesJSON, &entry.WarningCount,
		&entry.ImagesSucceeded, &entry.ImagesFailed, &entry.Status, &failedImagesJSON, &entry.CreatedAt,
	); err != nil {
		return entry, err
	}
	_ = json.Unmarshal([]byte(failedImagesJSON), &entry.FailedImages)
	_ = json.Unmarshal([]byte(selectedTypesJSON), &entry.SelectedTypes)
	return entry, nil
}

func (s *SQliteDB) DeleteDownloadHistoryEntry(ctx context.Context, id int64) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf("Delete Download History Entry %d", id), logging.LevelDebug)
	defer logAction.Complete()

	if s == nil || s.conn == nil {
		logAction.SetError("DB: connection is nil", "", map[string]any{})
		return *logAction.Error
	}

	_, err := s.conn.ExecContext(ctx, `DELETE FROM DownloadHistory WHERE id = ?;`, id)
	if err != nil {
		logAction.SetError("DB: DELETE DownloadHistory failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	return logging.LogErrorInfo{}
}

func (s *SQliteDB) DeleteAllDownloadHistory(ctx context.Context) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Delete All Download History", logging.LevelDebug)
	defer logAction.Complete()

	if s == nil || s.conn == nil {
		logAction.SetError("DB: connection is nil", "", map[string]any{})
		return *logAction.Error
	}

	_, err := s.conn.ExecContext(ctx, `DELETE FROM DownloadHistory;`)
	if err != nil {
		logAction.SetError("DB: DELETE ALL DownloadHistory failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	return logging.LogErrorInfo{}
}

// DeleteDownloadHistoryForSet removes an item's failures for a set that downloaded without any, and its failures before any set was downloaded
func (s *SQliteDB) DeleteDownloadHistoryForSet(ctx context.Context, tmdbID, libraryTitle, edition, setID string) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf("Delete Download History for Set %s", setID), logging.LevelDebug)
	defer logAction.Complete()

	if s == nil || s.conn == nil {
		logAction.SetError("DB: connection is nil", "", map[string]any{})
		return *logAction.Error
	}

	result, err := s.conn.ExecContext(ctx, `
DELETE FROM DownloadHistory
WHERE tmdb_id = ? AND library_title = ? AND edition = ? AND (set_id = ? OR set_id = '');
`, tmdbID, libraryTitle, edition, setID)
	if err != nil {
		logAction.SetError("DB: DELETE DownloadHistory for a set failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	if deleted, err := result.RowsAffected(); err == nil {
		logAction.AppendResult("deleted", deleted)
	}
	return logging.LogErrorInfo{}
}
