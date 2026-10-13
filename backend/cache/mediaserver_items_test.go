package cache

import (
	"aura/models"
	"testing"
)

func TestServerLibraryStores(t *testing.T) {
	t.Cleanup(func() { KeepServerLibraryStores(nil) })

	silo := ServerLibraryStore("silo")
	if ServerLibraryStore("silo") != silo {
		t.Error("the same media server should get the same cache")
	}
	if silo == LibraryStore || ServerLibraryStore("jellyfin") == silo {
		t.Error("each media server should have its own cache")
	}

	silo.UpdateSection(&models.LibrarySection{
		LibrarySectionBase: models.LibrarySectionBase{Title: "Movies"},
		MediaItems:         []models.MediaItem{{TMDB_ID: "1", Title: "Movie", RatingKey: "movie-tmdb-1"}},
	})
	if silo.GetItemsCount() != 1 || LibraryStore.GetItemsCount() != 0 {
		t.Errorf("items: silo %d, main %d, want 1 and 0", silo.GetItemsCount(), LibraryStore.GetItemsCount())
	}

	// Caches of media servers that are no longer used are removed
	KeepServerLibraryStores([]string{"jellyfin"})
	if ServerLibraryStore("silo").GetItemsCount() != 0 {
		t.Error("the removed media server's cache should be empty when it is used again")
	}
}
