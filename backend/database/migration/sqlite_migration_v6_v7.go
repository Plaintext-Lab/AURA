package migration

import (
	"aura/database"
	"aura/logging"
	"context"
)

// migrate_6_to_7 adds the LibraryGroups and LibraryGroupPolicies tables that
// power the linked-library feature.  Existing databases are unaffected – the
// new tables are empty until the user creates their first group.
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

	steps := []struct {
		name string
		sql  string
	}{
		{
			name: "LibraryGroups",
			sql: `
CREATE TABLE IF NOT EXISTS LibraryGroups (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	media_type TEXT NOT NULL CHECK (media_type IN ('movie','show')),
	library_ids TEXT NOT NULL DEFAULT '[]',
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);`,
		},
		{
			name: "LibraryGroupPolicies",
			sql: `
CREATE TABLE IF NOT EXISTS LibraryGroupPolicies (
	id TEXT PRIMARY KEY,
	group_id TEXT NOT NULL,
	tmdb_id TEXT NOT NULL,
	edition TEXT NOT NULL DEFAULT '',
	set_id TEXT NOT NULL,

	poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (poster_selected IN (0,1)),
	backdrop_selected INTEGER NOT NULL DEFAULT 0 CHECK (backdrop_selected IN (0,1)),
	season_poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (season_poster_selected IN (0,1)),
	special_season_poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (special_season_poster_selected IN (0,1)),
	titlecard_selected INTEGER NOT NULL DEFAULT 0 CHECK (titlecard_selected IN (0,1)),

	auto_download INTEGER NOT NULL DEFAULT 0 CHECK (auto_download IN (0,1)),
	future_updates_only INTEGER NOT NULL DEFAULT 0 CHECK (future_updates_only IN (0,1)),
	last_reconciled DATETIME,
	reconcile_status TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

	UNIQUE (group_id, tmdb_id, edition),

	FOREIGN KEY (group_id) REFERENCES LibraryGroups(id)
		ON DELETE CASCADE
		ON UPDATE CASCADE
);`,
		},
		{
			name: "idx_librarygrouppolicies_group_id",
			sql:  `CREATE INDEX IF NOT EXISTS idx_librarygrouppolicies_group_id ON LibraryGroupPolicies(group_id);`,
		},
		{
			name: "idx_librarygrouppolicies_tmdb_id",
			sql:  `CREATE INDEX IF NOT EXISTS idx_librarygrouppolicies_tmdb_id ON LibraryGroupPolicies(tmdb_id, edition);`,
		},
	}

	for _, step := range steps {
		if _, err := conn.ExecContext(ctx, step.sql); err != nil {
			logAction.SetError("Failed to execute migration step: "+step.name, err.Error(), map[string]any{"error": err.Error()})
			return *logAction.Error
		}
	}

	logging.LOGGER.Info().Timestamp().Msg("Database migration v6.0 to v7.0 completed successfully")
	return Err
}
