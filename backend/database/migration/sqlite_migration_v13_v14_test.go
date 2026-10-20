package migration

import (
	"aura/cache"
	"aura/config"
	"aura/database"
	"aura/logging"
	"aura/models"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// v13Schema is the three tables that gain library_id at v14 (without that column yet).
const v13Schema = `
CREATE TABLE VERSION (version INTEGER NOT NULL);
INSERT INTO VERSION (version) VALUES (13);
CREATE TABLE MediaItems (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	tmdb_id TEXT NOT NULL, library_title TEXT NOT NULL, edition TEXT NOT NULL DEFAULT '',
	rating_key TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL, year INTEGER NOT NULL,
	on_server INTEGER NOT NULL DEFAULT 0,
	first_seen_at INTEGER NOT NULL DEFAULT 0,
	last_seen_at INTEGER NOT NULL DEFAULT 0,
	UNIQUE (tmdb_id, library_title, edition)
);
CREATE TABLE SavedItems (
	tmdb_id TEXT NOT NULL, library_title TEXT NOT NULL, edition TEXT NOT NULL DEFAULT '',
	poster_set_id INTEGER NOT NULL, poster_selected INTEGER NOT NULL DEFAULT 0,
	no_auto_upgrade INTEGER NOT NULL DEFAULT 0,
	autodownload INTEGER NOT NULL DEFAULT 0, last_downloaded DATETIME NOT NULL,
	PRIMARY KEY (tmdb_id, library_title, edition, poster_set_id)
) WITHOUT ROWID;
CREATE TABLE IgnoredItems (
	tmdb_id TEXT NOT NULL, library_title TEXT NOT NULL, edition TEXT NOT NULL DEFAULT '',
	mode TEXT NOT NULL, current_sets TEXT NOT NULL DEFAULT '[]',
	PRIMARY KEY (tmdb_id, library_title, edition)
) WITHOUT ROWID;
INSERT INTO MediaItems (tmdb_id, library_title, rating_key, type, title, year, on_server) VALUES ('949', 'Movies', 'r1', 'movie', 'Heat', 1995, 1);
INSERT INTO SavedItems (tmdb_id, library_title, poster_set_id, poster_selected, autodownload, last_downloaded) VALUES ('949', 'Movies', 3, 1, 1, '2026-01-01');
INSERT INTO IgnoredItems (tmdb_id, library_title, mode, current_sets) VALUES ('1396', 'Movies', 'until-new-set-available', '["s1"]');
`

func openV13DB(t *testing.T) context.Context {
	t.Helper()
	dir := t.TempDir()
	config.ConfigPath = dir
	t.Setenv(config.DatabasePathEnv, "test.db")
	dbPath := filepath.Join(dir, "test.db")
	if err := os.WriteFile(dbPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(v13Schema); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	ctx = logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))
	database.Client = &database.SQliteDB{}
	t.Cleanup(func() { database.Client = nil })
	if _, Err := database.Client.Init(ctx); Err.Message != "" {
		t.Fatalf("Init: %s", Err.Message)
	}
	return ctx
}

func TestMigrate13To14KeepsDataAndAddsLibraryID(t *testing.T) {
	ctx := openV13DB(t)

	if Err := migrate_13_to_14(ctx); Err.Message != "" {
		t.Fatalf("migrate_13_to_14: %s", Err.Message)
	}

	conn, _, Err := database.GetDBConnection(ctx)
	if Err.Message != "" {
		t.Fatal(Err.Message)
	}
	defer conn.Close()

	var title, mode, sets, libraryID string
	var year, posterSelected, autodownload int
	if err := conn.QueryRow(`SELECT title, year, library_id FROM MediaItems WHERE tmdb_id = '949'`).Scan(&title, &year, &libraryID); err != nil {
		t.Fatal(err)
	}
	if title != "Heat" || year != 1995 || libraryID != "" {
		t.Errorf("MediaItems row = (%q, %d, %q)", title, year, libraryID)
	}
	if err := conn.QueryRow(`SELECT poster_selected, autodownload, library_id FROM SavedItems WHERE tmdb_id = '949'`).Scan(&posterSelected, &autodownload, &libraryID); err != nil {
		t.Fatal(err)
	}
	if posterSelected != 1 || autodownload != 1 || libraryID != "" {
		t.Errorf("SavedItems row = (%d, %d, %q)", posterSelected, autodownload, libraryID)
	}
	if err := conn.QueryRow(`SELECT mode, current_sets, library_id FROM IgnoredItems WHERE tmdb_id = '1396'`).Scan(&mode, &sets, &libraryID); err != nil {
		t.Fatal(err)
	}
	if mode != "until-new-set-available" || sets != `["s1"]` || libraryID != "" {
		t.Errorf("IgnoredItems row = (%q, %q, %q)", mode, sets, libraryID)
	}

	if Err := database.SyncLibraryRows(ctx, "Movies", "7"); Err.Message != "" {
		t.Fatalf("SyncLibraryRows: %s", Err.Message)
	}
	for _, table := range []string{"MediaItems", "SavedItems", "IgnoredItems"} {
		if err := conn.QueryRow(`SELECT library_id FROM ` + table).Scan(&libraryID); err != nil {
			t.Fatal(err)
		}
		if libraryID != "7" {
			t.Errorf("%s library_id = %q after backfill, want \"7\"", table, libraryID)
		}
	}
}

func TestMigrate13To14FillsLibraryIDFromAnEarlierScan(t *testing.T) {
	ctx := openV13DB(t)
	cache.LibraryStore.UpdateSection(&models.LibrarySection{LibrarySectionBase: models.LibrarySectionBase{ID: "7", Title: "Movies"}})
	t.Cleanup(cache.LibraryStore.ClearAllSections)

	if Err := migrate_13_to_14(ctx); Err.Message != "" {
		t.Fatalf("migrate_13_to_14: %s", Err.Message)
	}

	conn, _, Err := database.GetDBConnection(ctx)
	if Err.Message != "" {
		t.Fatal(Err.Message)
	}
	defer conn.Close()
	for _, table := range []string{"MediaItems", "SavedItems", "IgnoredItems"} {
		var libraryID string
		if err := conn.QueryRow(`SELECT library_id FROM ` + table).Scan(&libraryID); err != nil {
			t.Fatal(err)
		}
		if libraryID != "7" {
			t.Errorf("%s library_id = %q, want \"7\"", table, libraryID)
		}
	}
}

func TestMigrate13To14IsSafeWhenLibraryIDColumnExists(t *testing.T) {
	ctx := openV13DB(t)
	conn, _, Err := database.GetDBConnection(ctx)
	if Err.Message != "" {
		t.Fatal(Err.Message)
	}
	for _, table := range []string{"MediaItems", "SavedItems", "IgnoredItems"} {
		if _, err := conn.Exec(`ALTER TABLE ` + table + ` ADD COLUMN library_id TEXT NOT NULL DEFAULT 'legacy';`); err != nil {
			t.Fatal(err)
		}
	}

	if Err := migrate_13_to_14(ctx); Err.Message != "" {
		t.Fatalf("migrate_13_to_14: %s", Err.Message)
	}

	var libraryID string
	if err := conn.QueryRow(`SELECT library_id FROM MediaItems WHERE tmdb_id = '949'`).Scan(&libraryID); err != nil {
		t.Fatal(err)
	}
	if libraryID != "legacy" {
		t.Errorf("existing library_id = %q, want legacy value kept", libraryID)
	}
}
