package migration

import (
	"aura/cache"
	"aura/config"
	"aura/database"
	"aura/logging"
	"aura/models"
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// The v7 shape of the three tables that gain library_id.
const v7Schema = `
CREATE TABLE MediaItems (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	tmdb_id TEXT NOT NULL, library_title TEXT NOT NULL, edition TEXT NOT NULL DEFAULT '',
	rating_key TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL, year INTEGER NOT NULL,
	on_server INTEGER NOT NULL DEFAULT 0,
	UNIQUE (tmdb_id, library_title, edition)
);
CREATE TABLE SavedItems (
	tmdb_id TEXT NOT NULL, library_title TEXT NOT NULL, edition TEXT NOT NULL DEFAULT '',
	poster_set_id INTEGER NOT NULL, poster_selected INTEGER NOT NULL DEFAULT 0,
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

func openV7DB(t *testing.T) context.Context {
	t.Helper()
	dir := t.TempDir()
	config.ConfigPath = dir
	config.Current.Database = config.Config_Database{Type: "sqlite3", Path: "test.db"}
	raw, err := sql.Open("sqlite3", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(v7Schema); err != nil {
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

func TestMigrate7To8KeepsDataAndAddsLibraryID(t *testing.T) {
	ctx := openV7DB(t)

	// Running twice must be safe, e.g. after a failed version update.
	for range 2 {
		if Err := migrate_7_to_8(ctx); Err.Message != "" {
			t.Fatalf("migrate_7_to_8: %s", Err.Message)
		}
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

	// The first scan then fills the ID in.
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

// Upgrades from v0/v1 scan the libraries during migration, which also skips the
// start-up scan, so the migration has to fill IDs from that scan itself.
func TestMigrate7To8FillsLibraryIDFromAnEarlierScan(t *testing.T) {
	ctx := openV7DB(t)
	cache.LibraryStore.UpdateSection(&models.LibrarySection{LibrarySectionBase: models.LibrarySectionBase{ID: "7", Title: "Movies"}})
	t.Cleanup(cache.LibraryStore.ClearAllSections)

	if Err := migrate_7_to_8(ctx); Err.Message != "" {
		t.Fatalf("migrate_7_to_8: %s", Err.Message)
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
