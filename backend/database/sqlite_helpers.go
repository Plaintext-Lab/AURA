package database

import (
	"encoding/json"
	"time"
)

// parseJSONStringSlice deserialises a JSON array of strings (as stored in SQLite TEXT columns).
// Returns an empty slice when the input is empty or invalid.
func parseJSONStringSlice(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

// marshalJSONStringSlice serialises a string slice to a compact JSON array suitable for
// storing in a SQLite TEXT column.  Returns "[]" on error.
func marshalJSONStringSlice(s []string) string {
	if s == nil {
		s = []string{}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// parseDateTime parses a SQLite datetime string (either "YYYY-MM-DD HH:MM:SS" or RFC3339).
func parseDateTime(s string) (time.Time, error) {
	const sqliteLayout = "2006-01-02 15:04:05"
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse(sqliteLayout, s)
}

// nullableDateTime converts an optional *time.Time to an interface{} suitable for
// database/sql: nil if the pointer is nil, otherwise a formatted datetime string.
func nullableDateTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
