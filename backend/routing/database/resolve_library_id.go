package routes_db

import (
	"aura/config"
	"net/http"
)

// resolveLibraryID returns the stable library ID for use in DB identity operations.
// It first checks the "library_id" query parameter; if absent it falls back to
// "library_title" and looks up the matching library in the configured library list.
// If no configured library matches the title, the title itself is returned so that
// existing data (before the v6→v7 migration) can still be addressed.
func resolveLibraryID(r *http.Request) string {
	if id := r.URL.Query().Get("library_id"); id != "" {
		return id
	}
	title := r.URL.Query().Get("library_title")
	return libraryIDFromTitle(title)
}

// libraryIDFromTitle resolves a library_id from a display title by searching the
// configured library list. Falls back to the title itself if not found.
func libraryIDFromTitle(title string) string {
	for _, lib := range config.Current.MediaServer.Libraries {
		if lib.Title == title {
			return lib.ID
		}
	}
	return title
}

// resolveLibraryIDFromMediaItem ensures a MediaItem has a populated LibraryID.
// If LibraryID is already set it is returned unchanged; otherwise the LibraryTitle
// is used to look up the configured library ID.
func resolveLibraryIDFromMediaItem(libraryID, libraryTitle string) string {
	if libraryID != "" {
		return libraryID
	}
	return libraryIDFromTitle(libraryTitle)
}
