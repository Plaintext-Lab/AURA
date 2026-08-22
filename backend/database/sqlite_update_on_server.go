package database

import (
	"aura/logging"
	"context"
)

func (s *SQliteDB) UpdateMediaItemOnServer(ctx context.Context, tmdbID string, libraryID string, edition string, onServer bool) (logErr logging.LogErrorInfo) {

	logErr = logging.LogErrorInfo{}

	// Start a transaction
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		_, logAction := logging.AddSubActionToContext(ctx, "Updating MediaItem on_server flag in SQLite database", logging.LevelDebug)
		defer logAction.Complete()
		logAction.SetError("Failed to start transaction", "", map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	defer func() { _ = tx.Rollback() }()

	// Update the on_server flag for the specified MediaItem
	_, err = tx.ExecContext(ctx, `
		UPDATE MediaItems
		SET on_server = ?
		WHERE tmdb_id = ? AND library_id = ? AND edition = ?;
	`, onServer, tmdbID, libraryID, edition)
	if err != nil {
		_, logAction := logging.AddSubActionToContext(ctx, "Updating MediaItem on_server flag in SQLite database", logging.LevelDebug)
		defer logAction.Complete()
		logAction.SetError("Failed to update MediaItem on_server flag", "", map[string]any{"error": err.Error(), "tmdb_id": tmdbID, "library_id": libraryID, "edition": edition, "on_server": onServer})
		return *logAction.Error
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		_, logAction := logging.AddSubActionToContext(ctx, "Updating MediaItem on_server flag in SQLite database", logging.LevelDebug)
		defer logAction.Complete()
		logAction.SetError("Failed to commit transaction", "", map[string]any{"error": err.Error()})
		return *logAction.Error
	}

	return logErr
}
