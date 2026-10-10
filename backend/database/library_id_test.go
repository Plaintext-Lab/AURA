package database

import (
	"aura/cache"
	"aura/config"
	"aura/logging"
	"aura/models"
	"context"
	"database/sql"
	"testing"
	"time"
)

var libraryIDTables = []string{"MediaItems", "SavedItems", "IgnoredItems"}

// newTestDB creates a fresh install, the same way first start-up does.
func newTestDB(t *testing.T) (*SQliteDB, context.Context) {
	t.Helper()
	config.ConfigPath = t.TempDir()
	config.Current.Database = config.Config_Database{Type: "sqlite3", Path: "test.db"}
	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	ctx = logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))

	s := &SQliteDB{}
	if _, Err := s.Init(ctx); Err.Message != "" {
		t.Fatalf("Init: %s", Err.Message)
	}
	t.Cleanup(func() { s.conn.Close() })
	return s, ctx
}

func libraryIDOf(t *testing.T, conn *sql.DB, table, tmdbID string) string {
	t.Helper()
	var id string
	if err := conn.QueryRow("SELECT library_id FROM "+table+" WHERE tmdb_id = ?", tmdbID).Scan(&id); err != nil {
		t.Fatalf("%s: %v", table, err)
	}
	return id
}

func TestFreshInstallHasLibraryIDColumn(t *testing.T) {
	s, _ := newTestDB(t)
	for _, table := range libraryIDTables {
		var n int
		if err := s.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'library_id'", table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s has no library_id column", table)
		}
	}
}

func TestWritesStoreLibraryID(t *testing.T) {
	s, ctx := newTestDB(t)

	saved := models.DBSavedItem{
		MediaItem: models.MediaItem{
			TMDB_ID: "949", LibraryTitle: "Movies", LibraryID: "7", RatingKey: "101",
			Type: "movie", Title: "Heat", Year: 1995,
			Movie: &models.MediaItemMovie{File: models.MediaItemFile{Path: "/movies/Heat.mkv"}},
		},
		PosterSets: []models.DBPosterSetDetail{{
			PosterSet:      models.PosterSet{BaseSetInfo: models.BaseSetInfo{ID: "set1", Type: "movie", Title: "Heat", UserCreated: "u"}},
			LastDownloaded: time.Now(),
			SelectedTypes:  models.SelectedTypes{Poster: true},
		}},
	}
	if Err := s.UpsertSavedItem(ctx, saved); Err.Message != "" {
		t.Fatalf("UpsertSavedItem: %s", Err.Message)
	}
	for _, table := range []string{"MediaItems", "SavedItems"} {
		if got := libraryIDOf(t, s.conn, table, "949"); got != "7" {
			t.Errorf("%s library_id = %q, want \"7\"", table, got)
		}
	}

	// Reading back must return the stored ID.
	mediaItems, Err := s.GetAllMediaItems(ctx)
	if Err.Message != "" || len(mediaItems) != 1 {
		t.Fatalf("GetAllMediaItems: %d items, err=%s", len(mediaItems), Err.Message)
	}
	if mediaItems[0].LibraryID != "7" {
		t.Errorf("GetAllMediaItems LibraryID = %q, want \"7\"", mediaItems[0].LibraryID)
	}
	savedSets, Err := s.GetAllSavedSets(ctx, models.DBFilter{ItemsPerPage: 10, PageNumber: 1})
	if Err.Message != "" || len(savedSets.Items) != 1 {
		t.Fatalf("GetAllSavedSets: %d items, err=%s", len(savedSets.Items), Err.Message)
	}
	if savedSets.Items[0].MediaItem.LibraryID != "7" {
		t.Errorf("GetAllSavedSets LibraryID = %q, want \"7\"", savedSets.Items[0].MediaItem.LibraryID)
	}

	// The ignore route only sends the title, so the ID comes from the scanned section.
	cache.LibraryStore.UpdateSection(&models.LibrarySection{LibrarySectionBase: models.LibrarySectionBase{ID: "8", Title: "Shows"}})
	t.Cleanup(cache.LibraryStore.ClearAllSections)
	if Err := s.IgnoreMediaItem(ctx, "1396", "Shows", "", "always", ""); Err.Message != "" {
		t.Fatalf("IgnoreMediaItem: %s", Err.Message)
	}
	if got := libraryIDOf(t, s.conn, "IgnoredItems", "1396"); got != "8" {
		t.Errorf("IgnoredItems library_id = %q, want \"8\"", got)
	}
}

func TestBackfillLibraryIDOnlyFillsEmptyRowsForThatTitle(t *testing.T) {
	s, ctx := newTestDB(t)
	for _, q := range []string{
		`INSERT INTO MediaItems (tmdb_id, library_title, rating_key, type, title, year) VALUES ('1', 'Movies', 'r1', 'movie', 'A', 2000), ('2', 'Old', 'r2', 'movie', 'B', 2000)`,
		`INSERT INTO SavedItems (tmdb_id, library_title, poster_set_id, last_downloaded) VALUES ('1', 'Movies', 1, CURRENT_TIMESTAMP), ('2', 'Old', 1, CURRENT_TIMESTAMP)`,
		`INSERT INTO IgnoredItems (tmdb_id, library_title, mode) VALUES ('1', 'Movies', 'always'), ('2', 'Old', 'always')`,
	} {
		if _, err := s.conn.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	if Err := s.BackfillLibraryID(ctx, "Movies", "7"); Err.Message != "" {
		t.Fatalf("BackfillLibraryID: %s", Err.Message)
	}
	// A second run must not overwrite an ID that is already set.
	if Err := s.BackfillLibraryID(ctx, "Movies", "99"); Err.Message != "" {
		t.Fatalf("BackfillLibraryID: %s", Err.Message)
	}

	for _, table := range libraryIDTables {
		if got := libraryIDOf(t, s.conn, table, "1"); got != "7" {
			t.Errorf("%s Movies library_id = %q, want \"7\"", table, got)
		}
		if got := libraryIDOf(t, s.conn, table, "2"); got != "" {
			t.Errorf("%s unconfigured library_id = %q, want empty", table, got)
		}
	}
}
