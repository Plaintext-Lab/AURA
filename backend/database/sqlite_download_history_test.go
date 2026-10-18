package database

import (
	"aura/models"
	"testing"
)

func TestDownloadHistoryFailures(t *testing.T) {
	db := openTestMediaDB(t)
	ctx := testContext()

	entry := func(tmdbID, setID string) DownloadHistoryEntry {
		return DownloadHistoryEntry{
			TMDB_ID: tmdbID, LibraryTitle: "Series", MediaItemTitle: "Carrie", MediaItemYear: 2026,
			SetID: setID, SetTitle: "Carrie (2026) Set", SetCreator: "willtong93",
			SelectedTypes: models.SelectedTypes{Poster: true, Titlecard: true}, WarningCount: 2,
			ImagesSucceeded: 9, ImagesFailed: 1, FailedImages: []models.ImageDownloadResult{{ImageType: "titlecard", FailureReason: "timed out"}},
		}
	}
	for _, e := range []DownloadHistoryEntry{entry("1", "set"), entry("1", ""), entry("1", "other"), entry("2", "set")} {
		if Err := db.InsertDownloadHistoryEntry(ctx, e); Err.Message != "" {
			t.Fatal(Err.Message)
		}
	}

	entries, total, Err := db.GetDownloadHistory(ctx, 50, 0)
	if Err.Message != "" || total != 4 {
		t.Fatalf("GetDownloadHistory = %d %s, want 4", total, Err.Message)
	}
	got := entries[0]
	if got.SetCreator != "willtong93" || !got.SelectedTypes.Poster || !got.SelectedTypes.Titlecard || got.SelectedTypes.Backdrop || got.WarningCount != 2 || got.Status != "warning" {
		t.Errorf("entry = %+v, want the set's creator, selected types and warnings kept", got)
	}

	// The set downloading without failures clears its failures and the item's failures without a set
	if Err := db.DeleteDownloadHistoryForSet(ctx, "1", "Series", "", "set"); Err.Message != "" {
		t.Fatal(Err.Message)
	}
	entries, _, _ = db.GetDownloadHistory(ctx, 50, 0)
	left := map[string]bool{}
	for _, e := range entries {
		left[e.TMDB_ID+"|"+e.SetID] = true
	}
	if len(entries) != 2 || !left["1|other"] || !left["2|set"] {
		t.Errorf("left = %v, want the other set and the other item", left)
	}
}
