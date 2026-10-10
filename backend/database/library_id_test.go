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

func TestSyncLibraryRowsOnlyFillsEmptyRowsForThatTitle(t *testing.T) {
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

	if Err := s.SyncLibraryRows(ctx, "Movies", "7"); Err.Message != "" {
		t.Fatalf("SyncLibraryRows: %s", Err.Message)
	}
	// A second run must not overwrite an ID that is already set.
	if Err := s.SyncLibraryRows(ctx, "Movies", "99"); Err.Message != "" {
		t.Fatalf("SyncLibraryRows: %s", Err.Message)
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

func TestSyncLibraryRowsCarriesDataAcrossARename(t *testing.T) {
	s, ctx := newTestDB(t)
	for _, q := range []string{
		// Library 7 ("Movies", about to become "Films") has two editions of one movie.
		// Library 9 ("4K Movies") has the same movie.
		`INSERT INTO MediaItems (tmdb_id, library_title, library_id, edition, rating_key, type, title, year) VALUES
			('949', 'Movies', '7', '', 'r1', 'movie', 'Heat', 1995),
			('949', 'Movies', '7', 'Director''s Cut', 'r2', 'movie', 'Heat', 1995),
			('949', '4K Movies', '9', '', 'r3', 'movie', 'Heat', 1995)`,
		`INSERT INTO SavedItems (tmdb_id, library_title, library_id, edition, poster_set_id, last_downloaded) VALUES
			('949', 'Movies', '7', '', 1, CURRENT_TIMESTAMP),
			('949', 'Movies', '7', 'Director''s Cut', 2, CURRENT_TIMESTAMP),
			('949', '4K Movies', '9', '', 3, CURRENT_TIMESTAMP)`,
		`INSERT INTO IgnoredItems (tmdb_id, library_title, library_id, mode) VALUES ('1396', 'Movies', '7', 'always')`,
		`INSERT INTO PosterSets (id, set_id, type, title, user) VALUES (1, 's1', 'movie', 'A', 'u'), (2, 's2', 'movie', 'B', 'u'), (3, 's3', 'movie', 'C', 'u')`,
	} {
		if _, err := s.conn.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	if Err := s.SyncLibraryRows(ctx, "Films", "7"); Err.Message != "" {
		t.Fatalf("SyncLibraryRows: %s", Err.Message)
	}

	countByTitle := func(table, title string) int {
		var n int
		if err := s.conn.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE library_title = ?", title).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for table, want := range map[string]int{"MediaItems": 2, "SavedItems": 2, "IgnoredItems": 1} {
		if got := countByTitle(table, "Films"); got != want {
			t.Errorf("%s rows under the new title = %d, want %d", table, got, want)
		}
		if got := countByTitle(table, "Movies"); got != 0 {
			t.Errorf("%s rows left under the old title = %d, want 0", table, got)
		}
	}
	// The other library's copy keeps its own row and rating key.
	var ratingKey string
	if err := s.conn.QueryRow(`SELECT rating_key FROM MediaItems WHERE library_title = '4K Movies'`).Scan(&ratingKey); err != nil || ratingKey != "r3" {
		t.Errorf("4K Movies row = %q, %v; want r3", ratingKey, err)
	}

	// Lookups by the new title find the saved sets and the ignore.
	_, _, sets, Err := s.CheckIfMediaItemExists(ctx, "949", "Films", "Director's Cut")
	if Err.Message != "" || len(sets) != 1 {
		t.Errorf("saved sets for the renamed edition = %d, err=%s; want 1", len(sets), Err.Message)
	}
	ignored, _, _, Err := s.CheckIfMediaItemExists(ctx, "1396", "Films", "")
	if Err.Message != "" || !ignored {
		t.Errorf("ignored after rename = %v, err=%s; want true", ignored, Err.Message)
	}
}

func TestSyncLibraryRowsKeepsRowsThatWouldCollide(t *testing.T) {
	s, ctx := newTestDB(t)
	if _, err := s.conn.Exec(`INSERT INTO MediaItems (tmdb_id, library_title, library_id, rating_key, type, title, year) VALUES
		('949', 'Movies', '7', 'old', 'movie', 'Heat', 1995),
		('949', 'Films', '', 'new', 'movie', 'Heat', 1995)`); err != nil {
		t.Fatal(err)
	}

	if Err := s.SyncLibraryRows(ctx, "Films", "7"); Err.Message != "" {
		t.Fatalf("SyncLibraryRows: %s", Err.Message)
	}

	var n int
	if err := s.conn.QueryRow(`SELECT COUNT(*) FROM MediaItems WHERE tmdb_id = '949'`).Scan(&n); err != nil || n != 2 {
		t.Errorf("rows = %d, %v; want both kept", n, err)
	}
}
