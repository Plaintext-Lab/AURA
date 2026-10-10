package plex

import (
	"aura/config"
	"aura/logging"
	"aura/models"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const plexSectionItemsFixture = `{"MediaContainer":{"librarySectionID":7,"title1":"Movies","totalSize":1,"Metadata":[
	{"ratingKey":"101","type":"movie","title":"Heat","year":1995,"guid":"plex://movie/1",
	 "Guid":[{"id":"tmdb://949"}],"Media":[{"Part":[{"file":"/movies/Heat.mkv"}]}]}]}}`

func testContext() context.Context {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	return logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))
}

func TestGetLibrarySectionItemsSetsLibraryID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/library/sections/7/all" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(plexSectionItemsFixture))
	}))
	defer srv.Close()
	config.Current.MediaServer.URL = srv.URL

	section := models.LibrarySection{LibrarySectionBase: models.LibrarySectionBase{ID: "7", Title: "Movies", Type: "movie"}}
	items, _, Err := (&Plex{}).GetLibrarySectionItems(testContext(), section, "0", "")
	if Err.Message != "" {
		t.Fatalf("GetLibrarySectionItems: %s", Err.Message)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].LibraryID != "7" || items[0].LibraryTitle != "Movies" {
		t.Errorf("library = (%q, %q), want (\"7\", \"Movies\")", items[0].LibraryID, items[0].LibraryTitle)
	}
}

func TestExtractMediaItemFromResponseSetsLibraryID(t *testing.T) {
	metadata := PlexLibraryItemsMetadata{
		RatingKey:           "101",
		Type:                "movie",
		Title:               "Heat",
		Guids:               []PlexTagField{{ID: "tmdb://949"}},
		Media:               []PlexVideoMediaItem{{Part: []PlexVideoPartItem{{File: "/movies/Heat.mkv"}}}},
		LibrarySectionID:    7,
		LibrarySectionTitle: "Movies",
	}

	item, Err := extractMediaItemFromResponse(testContext(), metadata)
	if Err.Message != "" {
		t.Fatalf("extractMediaItemFromResponse: %s", Err.Message)
	}
	if item.LibraryID != "7" {
		t.Errorf("LibraryID = %q, want \"7\"", item.LibraryID)
	}
}
