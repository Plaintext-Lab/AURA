package migration

import (
	"aura/config"
	"aura/database"
	"aura/logging"
	"context"
)

// migrate_6_to_7 adds a "library_id" column to MediaItems, SavedItems and
// IgnoredItems and rebuilds the uniqueness constraints so that the stable
// library section ID is used for identity instead of the mutable library title.
//
// Backfill strategy: for every distinct library_title found in the old data we
// look up the corresponding library section ID from the current configuration.
// If no match is found the title is used as the id so that existing rows are
// never orphaned.
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

	libraryIDColumnExists, checkColumnErr := checkColumnExists(ctx, "MediaItems", "library_id")
	if checkColumnErr.Message != "" {
		return checkColumnErr
	}

	if !libraryIDColumnExists {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			logAction.SetError("Failed to begin transaction for adding library_id column", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}

		// --- MediaItems ---
		if _, err = tx.ExecContext(ctx, `ALTER TABLE MediaItems RENAME TO MediaItems_old;`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to rename MediaItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `
			CREATE TABLE MediaItems (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				tmdb_id TEXT NOT NULL,
				library_id TEXT NOT NULL DEFAULT '',
				library_title TEXT NOT NULL,
				edition TEXT NOT NULL DEFAULT '',
				rating_key TEXT NOT NULL,
				type TEXT NOT NULL CHECK (type IN ('movie','show')),
				title TEXT NOT NULL,
				year INTEGER NOT NULL,
				on_server INTEGER NOT NULL DEFAULT 0 CHECK (on_server IN (0,1)),
				UNIQUE (tmdb_id, library_id, edition)
			);
		`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to create new MediaItems table with library_id column", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO MediaItems (id, tmdb_id, library_id, library_title, edition, rating_key, type, title, year, on_server)
			SELECT id, tmdb_id, library_title, library_title, edition, rating_key, type, title, year, on_server
			FROM MediaItems_old;
		`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to copy data into new MediaItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `DROP TABLE MediaItems_old;`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to drop old MediaItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}

		// --- SavedItems ---
		if _, err = tx.ExecContext(ctx, `ALTER TABLE SavedItems RENAME TO SavedItems_old;`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to rename SavedItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `
			CREATE TABLE SavedItems (
				tmdb_id TEXT NOT NULL,
				library_id TEXT NOT NULL DEFAULT '',
				library_title TEXT NOT NULL,
				edition TEXT NOT NULL DEFAULT '',
				poster_set_id INTEGER NOT NULL,

				poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (poster_selected IN (0,1)),
				backdrop_selected INTEGER NOT NULL DEFAULT 0 CHECK (backdrop_selected IN (0,1)),
				season_poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (season_poster_selected IN (0,1)),
				special_season_poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (special_season_poster_selected IN (0,1)),
				titlecard_selected INTEGER NOT NULL DEFAULT 0 CHECK (titlecard_selected IN (0,1)),

				autodownload INTEGER NOT NULL DEFAULT 0 CHECK (autodownload IN (0,1)),
				auto_add_new_collection_items INTEGER NOT NULL DEFAULT 0 CHECK (auto_add_new_collection_items IN (0,1)),
				last_downloaded DATETIME NOT NULL,

				PRIMARY KEY (tmdb_id, library_id, edition, poster_set_id),

				FOREIGN KEY (poster_set_id) REFERENCES PosterSets(id)
					ON DELETE CASCADE
					ON UPDATE CASCADE,

				FOREIGN KEY (tmdb_id, library_id, edition) REFERENCES MediaItems(tmdb_id, library_id, edition)
					ON DELETE CASCADE
					ON UPDATE CASCADE
			) WITHOUT ROWID;
		`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to create new SavedItems table with library_id column", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO SavedItems (
				tmdb_id, library_id, library_title, edition, poster_set_id,
				poster_selected, backdrop_selected, season_poster_selected, special_season_poster_selected, titlecard_selected,
				autodownload, auto_add_new_collection_items, last_downloaded
			)
			SELECT
				tmdb_id, library_title, library_title, edition, poster_set_id,
				poster_selected, backdrop_selected, season_poster_selected, special_season_poster_selected, titlecard_selected,
				autodownload, auto_add_new_collection_items, last_downloaded
			FROM SavedItems_old;
		`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to copy data into new SavedItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `DROP TABLE SavedItems_old;`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to drop old SavedItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}

		// --- IgnoredItems ---
		if _, err = tx.ExecContext(ctx, `ALTER TABLE IgnoredItems RENAME TO IgnoredItems_old;`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to rename IgnoredItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `
			CREATE TABLE IgnoredItems (
				tmdb_id TEXT NOT NULL,
				library_id TEXT NOT NULL DEFAULT '',
				edition TEXT NOT NULL DEFAULT '',
				mode TEXT NOT NULL CHECK (mode IN ('always','until-set-available','until-new-set-available')),
				current_sets TEXT NOT NULL DEFAULT '[]',
				PRIMARY KEY (tmdb_id, library_id, edition)
			) WITHOUT ROWID;
		`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to create new IgnoredItems table with library_id column", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO IgnoredItems (tmdb_id, library_id, edition, mode, current_sets)
			SELECT tmdb_id, library_title, edition, mode, current_sets
			FROM IgnoredItems_old;
		`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to copy data into new IgnoredItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
		if _, err = tx.ExecContext(ctx, `DROP TABLE IgnoredItems_old;`); err != nil {
			tx.Rollback()
			logAction.SetError("Failed to drop old IgnoredItems table", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}

		if err = tx.Commit(); err != nil {
			logAction.SetError("Failed to commit transaction for adding library_id column", "", map[string]any{"error": err.Error()})
			return *logAction.Error
		}
	}

	// Backfill library_id for rows where library_id still matches library_title
	// (i.e. they were initialised with the fallback title value above).
	// We update each configured library individually so that we never need to
	// build unsafe string interpolation into SQL.
	for _, lib := range config.Current.MediaServer.Libraries {
		if lib.ID == "" || lib.Title == "" {
			continue
		}
		if _, err := conn.ExecContext(ctx,
			`UPDATE MediaItems SET library_id = ? WHERE library_title = ? AND library_id = library_title`,
			lib.ID, lib.Title,
		); err != nil {
			logAction.SetError("Failed to backfill library_id in MediaItems", "", map[string]any{"error": err.Error(), "library_title": lib.Title})
			return *logAction.Error
		}
		if _, err := conn.ExecContext(ctx,
			`UPDATE SavedItems SET library_id = ? WHERE library_title = ? AND library_id = library_title`,
			lib.ID, lib.Title,
		); err != nil {
			logAction.SetError("Failed to backfill library_id in SavedItems", "", map[string]any{"error": err.Error(), "library_title": lib.Title})
			return *logAction.Error
		}
		if _, err := conn.ExecContext(ctx,
			`UPDATE IgnoredItems SET library_id = ? WHERE library_id = ?`,
			lib.ID, lib.Title,
		); err != nil {
			logAction.SetError("Failed to backfill library_id in IgnoredItems", "", map[string]any{"error": err.Error(), "library_title": lib.Title})
			return *logAction.Error
		}
	}

	// Rebuild the index that query paths use for fast library_id look-ups.
	if _, err := conn.ExecContext(ctx, `DROP INDEX IF EXISTS idx_mediaitems_library_title`); err != nil {
		logAction.SetError("Failed to drop old library_title index on MediaItems", "", map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	if _, err := conn.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_mediaitems_library_id ON MediaItems (library_id)`); err != nil {
		logAction.SetError("Failed to create library_id index on MediaItems", "", map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	logging.LOGGER.Info().Timestamp().Msg("Database migration v6.0 to v7.0 completed successfully")
	return Err
}
