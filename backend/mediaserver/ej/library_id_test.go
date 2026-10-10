package ej

import (
	"aura/cache"
	"aura/config"
	"aura/logging"
	"aura/models"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	ejSectionItemsFixture = `{"TotalRecordCount":2,"Items":[
		{"Id":"201","Name":"Heat","Type":"Movie","ProductionYear":1995,"ProviderIds":{"Tmdb":"949"}},
		{"Id":"box1","Name":"Alien Collection","Type":"BoxSet"}]}`
	ejBoxSetItemsFixture = `{"TotalRecordCount":1,"Items":[
		{"Id":"202","Name":"Alien","Type":"Movie","ProductionYear":1979,"Path":"/movies/Alien.mkv","ProviderIds":{"Tmdb":"348"}}]}`
	ejItemDetailsFixture = `{"Id":"201","Name":"Heat","Type":"Movie","ProductionYear":1995,"Path":"/movies/Heat.mkv","ProviderIds":{"Tmdb":"949"}}`
)

var moviesSection = models.LibrarySection{LibrarySectionBase: models.LibrarySectionBase{
	ID: "lib1", Title: "Movies", Type: "movie", Paths: []string{"/movies"},
}}

func testContext() context.Context {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	return logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))
}

func startFakeServer(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/Users/u1/Items" && r.URL.Query().Get("ParentId") == "lib1":
			w.Write([]byte(ejSectionItemsFixture))
		case r.URL.Path == "/Users/u1/Items" && r.URL.Query().Get("ParentId") == "box1":
			w.Write([]byte(ejBoxSetItemsFixture))
		case r.URL.Path == "/Users/u1/Items/201":
			w.Write([]byte(ejItemDetailsFixture))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	config.Current.MediaServer.Type = "Jellyfin"
	config.Current.MediaServer.URL = srv.URL
	config.Current.MediaServer.UserID = "u1"
	config.Current.MediaServer.Libraries = []models.LibrarySection{moviesSection}
}

func TestGetLibrarySectionItemsSetsLibraryID(t *testing.T) {
	startFakeServer(t)

	items, _, Err := (&EJ{}).GetLibrarySectionItems(testContext(), moviesSection, "0", "")
	if Err.Message != "" {
		t.Fatalf("GetLibrarySectionItems: %s", Err.Message)
	}
	// One plain item plus one from splitting the BoxSet.
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	for _, item := range items {
		if item.LibraryID != "lib1" {
			t.Errorf("%s: LibraryID = %q, want \"lib1\"", item.Title, item.LibraryID)
		}
	}
}

func TestGetMediaItemDetailsResolvesLibraryIDFromSection(t *testing.T) {
	startFakeServer(t)
	cache.LibraryStore.UpdateSection(&moviesSection)
	t.Cleanup(cache.LibraryStore.ClearAllSections)

	item := &models.MediaItem{RatingKey: "201", LibraryTitle: "Movies"}
	found, Err := (&EJ{}).GetMediaItemDetails(testContext(), item)
	if Err.Message != "" || !found {
		t.Fatalf("GetMediaItemDetails: found=%v, err=%s", found, Err.Message)
	}
	if item.LibraryID != "lib1" {
		t.Errorf("LibraryID = %q, want \"lib1\"", item.LibraryID)
	}
}
