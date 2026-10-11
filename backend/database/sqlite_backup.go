package database

import (
	"aura/config"
	"aura/logging"
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

func (s *SQliteDB) Backup(ctx context.Context, currentVersion, newVersion int) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Backing up SQLite Database", logging.LevelInfo)
	defer logAction.Complete()

	Err = logging.LogErrorInfo{}

	// Get the DB path
	dbPath := config.DatabasePath()

	// Check if the database file exists
	// If it doesn't exist, there's nothing to back up
	// This can happen on first run
	// So we just return
	_, statErr := os.Stat(dbPath)
	if os.IsNotExist(statErr) {
		return Err
	} else if statErr != nil {
		logAction.SetError("Failed to stat database file for backup",
			"Ensure the database path is accessible.",
			map[string]any{
				"error": statErr.Error(),
				"path":  dbPath,
			})
		return *logAction.Error
	}

	// Create a backup file path with version and timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s_backup_v%d_to_v%d_%s.db", strings.TrimSuffix(dbPath, path.Ext(dbPath)), currentVersion, newVersion, timestamp)

	// Write the backup with VACUUM INTO instead of copying the database file.
	// In WAL mode, committed writes can still be in the WAL file, and copying the file while
	// another connection writes can produce a torn copy. VACUUM INTO writes a consistent
	// snapshot that includes the WAL contents, even while other connections are writing.
	actionBackupFile := logAction.AddSubAction("Backing up database file", logging.LevelTrace)
	_, err := s.conn.ExecContext(ctx, "VACUUM INTO ?", backupPath)
	if err != nil {
		actionBackupFile.SetError("Failed to back up database file",
			"Ensure the destination path is accessible and writable, and that there is sufficient disk space.",
			map[string]any{
				"error":       err.Error(),
				"source":      dbPath,
				"destination": backupPath,
			})
		return *actionBackupFile.Error
	}
	actionBackupFile.Complete()

	return Err
}
