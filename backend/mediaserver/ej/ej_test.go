package ej

import (
	"aura/config"
	"aura/logging"
	"aura/mediux"
	"aura/models"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

func testContext() context.Context {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	return logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))
}

func num(n int) *int { return &n }

// fakeJellyfin is a Jellyfin server that keeps the images uploaded to it, with a MediUX server for the images
type fakeJellyfin struct {
	mu        sync.Mutex
	requests  []string
	uploads   map[string][]byte // Path to the decoded image
	types     map[string]string // Path to the Content-Type
	backdrops []EmbyJellyItemImagesResponse
}

func newFakeJellyfin(t *testing.T) (*fakeJellyfin, *EJ) {
	t.Helper()
	f := &fakeJellyfin{uploads: map[string][]byte{}, types: map[string]string{}, backdrops: []EmbyJellyItemImagesResponse{{ImageType: "Backdrop", ImageIndex: 0, ImageTag: "old"}}}

	mediuxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("image " + r.URL.Path))
	}))
	t.Cleanup(mediuxServer.Close)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /System/Info", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		_, _ = w.Write([]byte(`{"Version":"10.9","ServerName":"Home Jellyfin"}`))
	})
	mux.HandleFunc("GET /Items/{key}/Images", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		f.mu.Lock()
		defer f.mu.Unlock()
		writeJSON(w, f.backdrops)
	})
	mux.HandleFunc("POST /Items/{key}/Images/{type}", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		body, _ := io.ReadAll(r.Body)
		decoded, _ := base64.StdEncoding.DecodeString(string(body))
		f.mu.Lock()
		defer f.mu.Unlock()
		f.uploads[r.URL.Path] = decoded
		f.types[r.URL.Path] = r.Header.Get("Content-Type")
		// The uploaded backdrop is added after the ones already there
		if r.PathValue("type") == "Backdrop" {
			f.backdrops = append(f.backdrops, EmbyJellyItemImagesResponse{ImageType: "Backdrop", ImageIndex: len(f.backdrops), ImageTag: "new"})
		}
	})
	mux.HandleFunc("POST /Items/{key}/Images/Backdrop/{index}/Index", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	previousConfig, previousMediux := config.Current, mediux.MediuxApiURL
	t.Cleanup(func() { config.Current, mediux.MediuxApiURL = previousConfig, previousMediux })
	config.Current.MainMediaServer = "main"
	config.Current.Images.CacheImages.Enabled = false
	mediux.MediuxApiURL = mediuxServer.URL

	return f, &EJ{Config: config.Config_MediaServer{ID: "other", Type: "Jellyfin", URL: server.URL, ApiToken: "jf-token"}}
}

func (f *fakeJellyfin) record(r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestEJTestConnection(t *testing.T) {
	_, e := newFakeJellyfin(t)
	ok, name, version, Err := e.TestConnection(testContext(), e.Config)
	if !ok || name != "Home Jellyfin" || version != "10.9" || Err.Message != "" {
		t.Errorf("TestConnection = %v, %q, %q, %s", ok, name, version, Err.Message)
	}
}

func TestEJDownloadApplyImage(t *testing.T) {
	fake, e := newFakeJellyfin(t)
	show := &models.MediaItem{Type: "show", Title: "Show", RatingKey: "200", Series: &models.MediaItemSeries{Seasons: []models.MediaItemSeason{
		{SeasonNumber: 1, RatingKey: "201", Episodes: []models.MediaItemEpisode{{SeasonNumber: 1, EpisodeNumber: 2, RatingKey: "203"}}},
	}}}

	// The poster, season poster and titlecard are uploaded to their own items
	for _, image := range []models.ImageFile{
		{ID: "poster", Type: "poster"},
		{ID: "season", Type: "season_poster", SeasonNumber: num(1)},
		{ID: "card", Type: "titlecard", SeasonNumber: num(1), EpisodeNumber: num(2)},
	} {
		if Err := e.DownloadApplyImageToMediaItem(testContext(), show, image); Err.Message != "" {
			t.Fatalf("applying the %s: %s", image.Type, Err.Message)
		}
	}
	for path, want := range map[string]string{
		"/Items/200/Images/Primary": "image /assets/poster",
		"/Items/201/Images/Primary": "image /assets/season",
		"/Items/203/Images/Primary": "image /assets/card",
	} {
		if string(fake.uploads[path]) != want || fake.types[path] != "image/png" {
			t.Errorf("%s got %q as %q, want %q as image/png", path, fake.uploads[path], fake.types[path], want)
		}
	}

	// A new backdrop is moved to the front
	if Err := e.DownloadApplyImageToMediaItem(testContext(), show, models.ImageFile{ID: "backdrop", Type: "backdrop"}); Err.Message != "" {
		t.Fatalf("applying the backdrop: %s", Err.Message)
	}
	if !slices.Contains(fake.requests, "POST /Items/200/Images/Backdrop/1/Index?newIndex=0") {
		t.Errorf("the new backdrop should be moved to index 0, requests = %v", fake.requests)
	}
}

func TestEJHelpers(t *testing.T) {
	jellyfin := AddEJAuthHeaders(config.Config_MediaServer{Type: "Jellyfin", ApiToken: "t"}, map[string]string{})
	if jellyfin["X-Emby-Token"] != "t" || jellyfin["Authorization"] != `MediaBrowser Token="t"` {
		t.Errorf("Jellyfin headers = %v", jellyfin)
	}
	if emby := AddEJAuthHeaders(config.Config_MediaServer{Type: "Emby", ApiToken: "t"}, map[string]string{}); emby["Authorization"] != "" {
		t.Errorf("Emby should only send X-Emby-Token, got %v", emby)
	}

	current := []EmbyJellyItemImagesResponse{{ImageType: "Primary", ImageTag: "p"}, {ImageType: "Backdrop", ImageIndex: 0, ImageTag: "a"}}
	updated := append(current, EmbyJellyItemImagesResponse{ImageType: "Backdrop", ImageIndex: 1, ImageTag: "b"})
	if got := findNewImage(testContext(), current, updated); got.ImageTag != "b" {
		t.Errorf("findNewImage = %+v, want the new backdrop", got)
	}
	if got := findNewImage(testContext(), current, current); got.ImageType != "" {
		t.Errorf("findNewImage with nothing new = %+v", got)
	}

	movie := models.MediaItem{Type: "movie", RatingKey: "1"}
	if got := getItemRatingKeyFromImageFile(movie, models.ImageFile{Type: "poster"}); got != "1" {
		t.Errorf("movie poster key = %q", got)
	}
}
