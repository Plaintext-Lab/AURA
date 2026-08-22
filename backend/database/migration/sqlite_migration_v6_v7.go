package migration

import (
	"aura/database"
	"aura/logging"
	"context"
)

// migrate_6_to_7 creates the ActivitySummaries table for caching aggregate
// activity data from Tautulli / Tracearr. No existing tables are modified.
func migrate_6_to_7(ctx context.Context) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Migrating Database from v6 to v7", logging.LevelInfo)
	defer logAction.Complete()
	logging.LOGGER.Info().Timestamp().Int("From Version", 6).Int("To Version", 7).Msg("Starting database migration")

	Err = logging.LogErrorInfo{}

	backupErr := database.Backup(ctx, 6, 7)
	if backupErr.Message != "" {
		return backupErr
	}

	conn, _, getDBConnErr := database.GetDBConnection(ctx)
	if getDBConnErr.Message != "" {
		return getDBConnErr
	}

	query := `
CREATE TABLE IF NOT EXISTS ActivitySummaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source            TEXT NOT NULL,
    library_id        TEXT NOT NULL,
    rating_key        TEXT NOT NULL,
    grandparent_key   TEXT NOT NULL DEFAULT '',
    media_type        TEXT NOT NULL CHECK (media_type IN ('movie','show','episode')),
    tmdb_id           TEXT NOT NULL DEFAULT '',
    tvdb_id           TEXT NOT NULL DEFAULT '',
    title             TEXT NOT NULL DEFAULT '',
    play_count        INTEGER NOT NULL DEFAULT 0,
    watch_time_secs   INTEGER NOT NULL DEFAULT 0,
    last_watched      DATETIME,
    window_start      DATETIME NOT NULL,
    window_end        DATETIME NOT NULL,
    synced_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source, library_id, rating_key)
);
CREATE INDEX IF NOT EXISTS idx_activitysummaries_source ON ActivitySummaries(source);
CREATE INDEX IF NOT EXISTS idx_activitysummaries_tmdb ON ActivitySummaries(tmdb_id);
CREATE INDEX IF NOT EXISTS idx_activitysummaries_play_count ON ActivitySummaries(play_count DESC);
`
	_, err := conn.ExecContext(ctx, query)
	if err != nil {
		logAction.SetError("Failed to create ActivitySummaries table", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	logging.LOGGER.Info().Timestamp().Msg("Database migration v6.0 to v7.0 completed successfully")
	return Err
}
