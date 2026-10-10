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

// BackfillLibraryID runs at scan time rather than in a migration, because the
// configured library list holds only titles and section IDs are unknown until a scan.
func (s *SQliteDB) BackfillLibraryID(ctx context.Context, libraryTitle, libraryID string) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf("Backfilling library_id for '%s'", libraryTitle), logging.LevelDebug)
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

	var updated int64
	for _, table := range []string{"MediaItems", "SavedItems", "IgnoredItems"} {
		res, err := tx.ExecContext(ctx, `UPDATE `+table+` SET library_id = ? WHERE library_title = ? AND library_id = '';`, libraryID, libraryTitle)
		if err != nil {
			logAction.SetError("DB: backfill library_id failed", err.Error(), map[string]any{"error": err.Error(), "table": table})
			return *logAction.Error
		}
		n, _ := res.RowsAffected()
		updated += n
	}

	if err := tx.Commit(); err != nil {
		logAction.SetError("DB: TX COMMIT failed", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	logAction.AppendResult("rows_updated", updated)
	return logging.LogErrorInfo{}
}
