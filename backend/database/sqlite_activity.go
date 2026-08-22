package database

import (
	"aura/logging"
	"context"
	"database/sql"
	"time"
)

func (s *SQliteDB) UpsertActivitySummaries(ctx context.Context, summaries []ActivitySummaryRow) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Upsert Activity Summaries", logging.LevelTrace)
	defer logAction.Complete()

	if len(summaries) == 0 {
		return logging.LogErrorInfo{}
	}

	conn, _, getErr := s.GetDBConnection(ctx)
	if getErr.Message != "" {
		return getErr
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		logAction.SetError("Failed to begin transaction", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO ActivitySummaries
    (source, library_id, rating_key, grandparent_key, media_type, tmdb_id, tvdb_id, title,
     play_count, watch_time_secs, last_watched, window_start, window_end, synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(source, library_id, rating_key) DO UPDATE SET
    grandparent_key  = excluded.grandparent_key,
    media_type       = excluded.media_type,
    tmdb_id          = excluded.tmdb_id,
    tvdb_id          = excluded.tvdb_id,
    title            = excluded.title,
    play_count       = excluded.play_count,
    watch_time_secs  = excluded.watch_time_secs,
    last_watched     = excluded.last_watched,
    window_start     = excluded.window_start,
    window_end       = excluded.window_end,
    synced_at        = excluded.synced_at
`)
	if err != nil {
		tx.Rollback()
		logAction.SetError("Failed to prepare upsert statement", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)

	for _, row := range summaries {
		_, err = stmt.ExecContext(ctx,
			row.Source, row.LibraryID, row.RatingKey, row.GrandparentKey, row.MediaType,
			row.TmdbID, row.TvdbID, row.Title,
			row.PlayCount, row.WatchTimeSecs, row.LastWatched,
			row.WindowStart, row.WindowEnd, now,
		)
		if err != nil {
			tx.Rollback()
			logAction.SetError("Failed to upsert activity summary row", err.Error(), map[string]any{"error": err.Error()})
			return *logAction.Error
		}
	}

	if err = tx.Commit(); err != nil {
		logAction.SetError("Failed to commit activity summaries transaction", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	return logging.LogErrorInfo{}
}

func (s *SQliteDB) GetActivitySummaries(ctx context.Context) (rows []ActivitySummaryRow, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Get Activity Summaries", logging.LevelTrace)
	defer logAction.Complete()

	conn, _, getErr := s.GetDBConnection(ctx)
	if getErr.Message != "" {
		return nil, getErr
	}

	query := `
SELECT id, source, library_id, rating_key, grandparent_key, media_type, tmdb_id, tvdb_id, title,
       play_count, watch_time_secs, last_watched, window_start, window_end, synced_at
FROM ActivitySummaries
ORDER BY play_count DESC
`
	dbRows, err := conn.QueryContext(ctx, query)
	if err != nil {
		logAction.SetError("Failed to query activity summaries", err.Error(), map[string]any{"error": err.Error()})
		return nil, *logAction.Error
	}
	defer dbRows.Close()

	for dbRows.Next() {
		var r ActivitySummaryRow
		var lastWatched sql.NullString
		err = dbRows.Scan(
			&r.ID, &r.Source, &r.LibraryID, &r.RatingKey, &r.GrandparentKey, &r.MediaType,
			&r.TmdbID, &r.TvdbID, &r.Title,
			&r.PlayCount, &r.WatchTimeSecs, &lastWatched,
			&r.WindowStart, &r.WindowEnd, &r.SyncedAt,
		)
		if err != nil {
			logAction.SetError("Failed to scan activity summary row", err.Error(), map[string]any{"error": err.Error()})
			return nil, *logAction.Error
		}
		if lastWatched.Valid {
			r.LastWatched = lastWatched.String
		}
		rows = append(rows, r)
	}
	if err = dbRows.Err(); err != nil {
		logAction.SetError("Error iterating activity summary rows", err.Error(), map[string]any{"error": err.Error()})
		return nil, *logAction.Error
	}

	return rows, logging.LogErrorInfo{}
}

func (s *SQliteDB) DeleteActivitySummaries(ctx context.Context, source string) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Delete Activity Summaries", logging.LevelTrace)
	defer logAction.Complete()

	conn, _, getErr := s.GetDBConnection(ctx)
	if getErr.Message != "" {
		return getErr
	}

	_, err := conn.ExecContext(ctx, `DELETE FROM ActivitySummaries WHERE source = ?`, source)
	if err != nil {
		logAction.SetError("Failed to delete activity summaries", err.Error(), map[string]any{"error": err.Error(), "source": source})
		return *logAction.Error
	}

	return logging.LogErrorInfo{}
}
