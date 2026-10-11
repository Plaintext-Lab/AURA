package database

import (
	"context"
	"database/sql"
	"fmt"
)

// Config migrations update stored values when a config field is renamed, moved or changes type.
// Adding or removing a field doesn't need a migration.
// TestConfigSchemaSnapshot fails when a field changes without one.
//
// To add a migration, append it to configMigrations:
//
//	{
//		Description: "Rename MediaServer.Token to MediaServer.ApiToken",
//		Migrate:     renameConfigKey("MediaServer.Token", "MediaServer.ApiToken"),
//	},
type configMigration struct {
	Description string
	Migrate     func(ctx context.Context, tx *sql.Tx) error
}

// configMigrations[0] migrates from version 1 to version 2, and so on
var configMigrations = []configMigration{}

// Version 1 is the first config schema, stored without any migrations
var LATEST_CONFIG_SCHEMA_VERSION = len(configMigrations) + 1

// renameConfigKey moves a stored value to a new key. The value is kept as is.
func renameConfigKey(oldKey, newKey string) func(ctx context.Context, tx *sql.Tx) error {
	return func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
UPDATE OR REPLACE Config SET key = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?;`, newKey, oldKey)
		if err != nil {
			return fmt.Errorf("failed to rename config key '%s' to '%s': %w", oldKey, newKey, err)
		}
		return nil
	}
}
