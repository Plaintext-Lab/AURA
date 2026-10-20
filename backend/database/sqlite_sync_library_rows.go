package database

import (
	"aura/cache"
	"aura/logging"
	"context"
	"fmt"
)

// libraryIDForTitle keeps a known ID, otherwise takes it from the scanned section with that title.
func libraryIDForTitle(libraryTitle, libraryID string) string {
	if libraryID != "" {
		return libraryID
	}
	if section, ok := cache.LibraryStore.GetSectionByTitle(libraryTitle); ok {
		return section.ID
	}
	return ""
}

// SyncLibraryRows lines stored rows up with a scanned library: rows that carry its
// ID take its current title, so saved sets and ignores survive a rename, and rows
// under its title with no ID yet get the ID. It runs at scan time because the
// configured library list holds only titles and section IDs are unknown until then.
func (s *SQliteDB) SyncLibraryRows(ctx context.Context, libraryTitle, libraryID string) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf("Syncing stored rows for library '%s'", libraryTitle), logging.LevelDebug)
	defer logAction.Complete()

	if s == nil || s.conn == nil {
		logAction.SetError("DB: connection is nil", "", map[string]any{})
		return *logAction.Error
	}
	if libraryTitle == "" || libraryID == "" {
		return logging.LogErrorInfo{}
	}

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		logAction.SetError("DB: TX BEGIN failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	defer func() { _ = tx.Rollback() }()

	var renamed, backfilled int64
	for _, table := range []string{"MediaItems", "SavedItems", "IgnoredItems"} {
		// OR IGNORE keeps a row under its old title if the new title already has the same item.
		res, err := tx.ExecContext(ctx, `UPDATE OR IGNORE `+table+` SET library_title = ? WHERE library_id = ? AND library_title <> ?;`, libraryTitle, libraryID, libraryTitle)
		if err != nil {
			logAction.SetError("DB: sync library_title failed", err.Error(), map[string]any{"error": err.Error(), "table": table})
			return *logAction.Error
		}
		n, _ := res.RowsAffected()
		renamed += n

		res, err = tx.ExecContext(ctx, `UPDATE `+table+` SET library_id = ? WHERE library_title = ? AND library_id = '';`, libraryID, libraryTitle)
		if err != nil {
			logAction.SetError("DB: backfill library_id failed", err.Error(), map[string]any{"error": err.Error(), "table": table})
			return *logAction.Error
		}
		n, _ = res.RowsAffected()
		backfilled += n
	}

	if err := tx.Commit(); err != nil {
		logAction.SetError("DB: TX COMMIT failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	logAction.AppendResult("rows_renamed", renamed)
	logAction.AppendResult("rows_backfilled", backfilled)
	return logging.LogErrorInfo{}
}
