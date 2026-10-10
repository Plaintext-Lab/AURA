package migration

import (
	"aura/cache"
	"aura/database"
	"aura/logging"
	"context"
)

// migrate_7_to_8 adds an empty library_id column to MediaItems, SavedItems and
// IgnoredItems. The configured libraries hold only titles, so IDs are normally
// written by database.SyncLibraryRows during the next scan. Upgrades from v0/v1
// scan during migration and skip that start-up scan, so fill from any scanned
// sections here too.
func migrate_7_to_8(ctx context.Context) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Migrating Database from v7 to v8", logging.LevelInfo)
	defer logAction.Complete()
	logging.LOGGER.Info().Timestamp().Int("From Version", 7).Int("To Version", 8).Msg("Starting database migration")

	Err = logging.LogErrorInfo{}

	backupErr := database.Backup(ctx, 7, 8)
	if backupErr.Message != "" {
		return backupErr
	}

	conn, _, getDBConnErr := database.GetDBConnection(ctx)
	if getDBConnErr.Message != "" {
		return getDBConnErr
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		logAction.SetError("Failed to begin transaction for adding library_id column", "", map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	for _, table := range []string{"MediaItems", "SavedItems", "IgnoredItems"} {
		exists, checkColumnErr := checkColumnExists(ctx, table, "library_id")
		if checkColumnErr.Message != "" {
			tx.Rollback()
			return checkColumnErr
		}
		if exists {
			continue
		}
		if _, err = tx.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN library_id TEXT NOT NULL DEFAULT '';`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to add library_id column to "+table, "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
	}

	if err = tx.Commit(); err != nil {
		logAction.SetError("Failed to commit transaction for adding library_id column", "", map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	for _, section := range cache.LibraryStore.GetAllSectionsSortedByTitle() {
		if syncErr := database.SyncLibraryRows(ctx, section.Title, section.ID); syncErr.Message != "" {
			return syncErr
		}
	}

	logging.LOGGER.Info().Timestamp().Msg("Database migration v7.0 to v8.0 completed successfully")
	return Err
}
