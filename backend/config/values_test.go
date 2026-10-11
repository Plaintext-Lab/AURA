package config

import (
	"aura/models"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var updateConfigSchema = flag.Bool("update", false, "update the config schema snapshot (testdata/config_schema.golden)")

// testConfig returns a config with every kind of field set
func testConfig() Config {
	cfg := DefaultConfig()
	cfg.Auth.Enabled = true
	cfg.Auth.APIKeyHash = "$argon2id$hash"
	cfg.Auth.AllowedOrigins = []string{"https://a.example.com", "https://b.example.com"}
	cfg.Auth.OIDC.ClientSecret = "secret: with \"quotes\" and\nnew lines"
	cfg.MediaServer = Config_MediaServer{
		Type:     "Plex",
		URL:      "http://10.1.1.100:32400",
		ApiToken: "12345",
		UserID:   "not-saved",
		Libraries: []models.LibrarySection{
			{LibrarySectionBase: models.LibrarySectionBase{ID: "1", Title: "Movies", Type: "movie"}},
		},
		EnablePlexEventListener: true,
	}
	disabled := false
	cfg.Jobs.AutoDownload.Enabled = &disabled
	cfg.Jobs.CheckMediuxSiteLink.Enabled = nil
	cfg.TMDB.ApiToken = "tmdb-token"
	cfg.Notifications.Providers = []Config_Notification_Provider{
		{Provider: "Discord", Enabled: true, Discord: &Config_Notification_Discord{Webhook: "https://discord.example.com/hook"}},
		{Provider: "Webhook", Webhook: &Config_Notification_Webhook{URL: "https://hook.example.com", Headers: map[string]string{"Authorization": "Bearer 1"}}},
	}
	cfg.SonarrRadarr.Applications = []Config_SonarrRadarrApp{
		{Type: "sonarr", Library: "TV Shows", URL: "http://sonarr:8989", ApiToken: "abc"},
	}
	cfg.UserPreferences.DownloadDefaults = []string{"poster", "titlecard"}
	cfg.UserPreferences.DefaultAddToQueue = true
	return cfg
}

func TestConfigValuesRoundTrip(t *testing.T) {
	cfg := testConfig()
	values, err := cfg.ToValues()
	if err != nil {
		t.Fatal(err)
	}

	if _, found := values["MediaServer.UserID"]; found {
		t.Error("MediaServer.UserID should not be stored")
	}
	if got := values["MediaServer.URL"]; got != `"http://10.1.1.100:32400"` {
		t.Errorf("MediaServer.URL = %s", got)
	}

	loaded, decodeErrors, unknownKeys := ConfigFromValues(values)
	if len(decodeErrors) > 0 || len(unknownKeys) > 0 {
		t.Fatalf("decodeErrors = %v, unknownKeys = %v", decodeErrors, unknownKeys)
	}

	// The User ID is not stored, so it is empty after loading
	cfg.MediaServer.UserID = ""
	if !reflect.DeepEqual(cfg, loaded) {
		t.Errorf("config changed after saving and loading\nsaved:  %+v\nloaded: %+v", cfg, loaded)
	}
}

func TestConfigFromValuesUsesDefaults(t *testing.T) {
	values, err := testConfig().ToValues()
	if err != nil {
		t.Fatal(err)
	}

	// A missing value uses the default value
	delete(values, "Mediux.DownloadQuality")
	// A value that doesn't decode uses the default value
	values["Logging.Level"] = `{"not": "a string"}`
	values["MediaServer.EnablePlexEventListener"] = `not json`
	// A value that is no longer used is reported
	values["Removed.Field"] = `"old"`

	loaded, decodeErrors, unknownKeys := ConfigFromValues(values)
	defaults := DefaultConfig()

	if loaded.Mediux.DownloadQuality != defaults.Mediux.DownloadQuality {
		t.Errorf("Mediux.DownloadQuality = %q, want default %q", loaded.Mediux.DownloadQuality, defaults.Mediux.DownloadQuality)
	}
	if loaded.Logging.Level != defaults.Logging.Level {
		t.Errorf("Logging.Level = %q, want default %q", loaded.Logging.Level, defaults.Logging.Level)
	}
	if loaded.MediaServer.EnablePlexEventListener != defaults.MediaServer.EnablePlexEventListener {
		t.Errorf("MediaServer.EnablePlexEventListener = %v, want default", loaded.MediaServer.EnablePlexEventListener)
	}
	if _, found := decodeErrors["Logging.Level"]; !found {
		t.Error("expected a decode error for Logging.Level")
	}
	if _, found := decodeErrors["MediaServer.EnablePlexEventListener"]; !found {
		t.Error("expected a decode error for MediaServer.EnablePlexEventListener")
	}
	if len(unknownKeys) != 1 || unknownKeys[0] != "Removed.Field" {
		t.Errorf("unknownKeys = %v, want [Removed.Field]", unknownKeys)
	}
	// Other values still load
	if loaded.MediaServer.URL != "http://10.1.1.100:32400" {
		t.Errorf("MediaServer.URL = %q", loaded.MediaServer.URL)
	}
}

// TestConfigSchemaSnapshot fails when a stored config field changes.
// Added or removed fields: update the snapshot.
// Renamed, moved or retyped fields: add a config migration first, then update the snapshot:
//
//	go test ./config -run TestConfigSchemaSnapshot -update
func TestConfigSchemaSnapshot(t *testing.T) {
	goldenFile := filepath.Join("testdata", "config_schema.golden")
	current := configSchema()

	if *updateConfigSchema {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, []byte(strings.Join(current, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	data, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("reading %s: %v (run with -update to create it)", goldenFile, err)
	}
	snapshot := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		key, fieldType, _ := strings.Cut(line, ": ")
		snapshot[key] = fieldType
	}
	currentTypes := map[string]string{}
	for _, line := range current {
		key, fieldType, _ := strings.Cut(line, ": ")
		currentTypes[key] = fieldType
	}

	var added, removed, changed []string
	for key, fieldType := range currentTypes {
		snapshotType, found := snapshot[key]
		switch {
		case !found:
			added = append(added, fmt.Sprintf("%s: %s", key, fieldType))
		case snapshotType != fieldType:
			changed = append(changed, fmt.Sprintf("%s: %s -> %s", key, snapshotType, fieldType))
		}
	}
	for key, fieldType := range snapshot {
		if _, found := currentTypes[key]; !found {
			removed = append(removed, fmt.Sprintf("%s: %s", key, fieldType))
		}
	}
	if len(added) == 0 && len(removed) == 0 && len(changed) == 0 {
		return
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)

	t.Errorf(`The stored config schema changed.

Added (safe, existing installs use the default value):
  %s
Removed (stored values are ignored; if this is a rename or move, add a config migration):
  %s
Changed type (add a config migration that converts the stored value):
  %s

Config migrations are in database/sqlite_config_migrations.go. Then update the snapshot:
  go test ./config -run TestConfigSchemaSnapshot -update`,
		strings.Join(added, "\n  "), strings.Join(removed, "\n  "), strings.Join(changed, "\n  "))
}

// configSchema returns every stored config key with its type, sorted.
// Fields inside lists and maps use "[]" and "{}" in the key.
func configSchema() []string {
	lines := []string{}
	collectConfigSchema(reflect.TypeOf(Config{}), "", &lines)
	sort.Strings(lines)
	return lines
}

func collectConfigSchema(t reflect.Type, prefix string, lines *[]string) {
	for i := 0; i < t.NumField(); i++ {
		structField := t.Field(i)
		name := yamlFieldName(structField)
		if !structField.IsExported() || name == "-" {
			continue
		}
		key := joinConfigKey(prefix, name)
		if configValuesExcludedKeys[key] {
			continue
		}
		if structField.Type.Kind() == reflect.Struct {
			collectConfigSchema(structField.Type, key, lines)
			continue
		}
		collectConfigSchemaValue(structField.Type, key, lines)
	}
}

// collectConfigSchemaValue adds a stored value, expanding config structs inside lists, maps and pointers
func collectConfigSchemaValue(t reflect.Type, key string, lines *[]string) {
	switch {
	case t.Kind() == reflect.Pointer && isConfigStruct(t.Elem()):
		*lines = append(*lines, key+": object")
		collectConfigSchema(t.Elem(), key, lines)
	case t.Kind() == reflect.Slice && isConfigStruct(elemStruct(t.Elem())):
		*lines = append(*lines, key+": list")
		collectConfigSchemaValue(t.Elem(), key+"[]", lines)
	case t.Kind() == reflect.Map && isConfigStruct(elemStruct(t.Elem())):
		*lines = append(*lines, key+": map["+t.Key().String()+"]")
		collectConfigSchemaValue(t.Elem(), key+"{}", lines)
	case isConfigStruct(t):
		collectConfigSchema(t, key, lines)
	default:
		*lines = append(*lines, key+": "+t.String())
	}
}

func elemStruct(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}

// isConfigStruct reports whether t is a struct from the config package
func isConfigStruct(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && t.PkgPath() == reflect.TypeOf(Config{}).PkgPath()
}
