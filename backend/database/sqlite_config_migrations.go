package database

import (
	"aura/config"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
var configMigrations = []configMigration{
	{
		Description: "Move MediaServer to MediaServers, as the main media server",
		Migrate:     moveMediaServerToMediaServers,
	},
}

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

// moveMediaServerToMediaServers moves the MediaServer values into the MediaServers list, as the main media server
func moveMediaServerToMediaServers(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT key, value FROM Config WHERE key LIKE 'MediaServer.%';`)
	if err != nil {
		return fmt.Errorf("failed to get MediaServer values: %w", err)
	}
	server := map[string]json.RawMessage{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			rows.Close()
			return fmt.Errorf("failed to read MediaServer value: %w", err)
		}
		server[strings.TrimPrefix(key, "MediaServer.")] = json.RawMessage(value)
	}
	rows.Close()

	// Only add the media server if one was set up
	mediaServers, mainMediaServer := "null", `""`
	var serverType string
	_ = json.Unmarshal(server["Type"], &serverType)
	if serverType != "" {
		id, _ := json.Marshal(config.NewMediaServerID())
		server["ID"] = id
		server["Name"] = server["Type"]
		server["Enabled"] = json.RawMessage("true")
		list, err := json.Marshal([]map[string]json.RawMessage{server})
		if err != nil {
			return fmt.Errorf("failed to build MediaServers: %w", err)
		}
		mediaServers, mainMediaServer = string(list), string(id)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT OR REPLACE INTO Config (key, value, updated_at) VALUES
    ('MediaServers', ?, CURRENT_TIMESTAMP),
    ('MainMediaServer', ?, CURRENT_TIMESTAMP);`, mediaServers, mainMediaServer); err != nil {
		return fmt.Errorf("failed to save MediaServers: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM Config WHERE key LIKE 'MediaServer.%';`); err != nil {
		return fmt.Errorf("failed to remove MediaServer values: %w", err)
	}
	return nil
}
