package downloadqueue

import (
	"aura/database"
	"aura/logging"
	"aura/models"
	"context"
	"testing"
)

func TestRetryFailedDownload(t *testing.T) {
	ctx, ld := logging.CreateLoggingContext(context.Background(), "test")
	ctx = logging.WithCurrentAction(ctx, ld.AddAction("test", logging.LevelInfo))

	entries := map[int64]database.DownloadHistoryEntry{
		1: {ID: 1, TMDB_ID: "100", LibraryTitle: "Movies", MediaItemTitle: "Hope", SetID: "a"},
		2: {ID: 2, TMDB_ID: "100", LibraryTitle: "Movies", MediaItemTitle: "Hope"},
		3: {ID: 3, TMDB_ID: "100", LibraryTitle: "Movies", MediaItemTitle: "Hope", SetID: "gone"},
	}
	set := func(id string) models.DBPosterSetDetail {
		return models.DBPosterSetDetail{PosterSet: models.PosterSet{BaseSetInfo: models.BaseSetInfo{ID: id}}}
	}
	jobs := []database.DownloadQueueJob{}
	var queued []models.DBSavedItem

	getHistoryEntry = func(_ context.Context, id int64) (database.DownloadHistoryEntry, bool, logging.LogErrorInfo) {
		entry, ok := entries[id]
		return entry, ok, logging.LogErrorInfo{}
	}
	getSavedSets = func(_ context.Context, _ models.DBFilter) (database.PagedSavedItems, logging.LogErrorInfo) {
		return database.PagedSavedItems{Items: []models.DBSavedItem{
			{MediaItem: models.MediaItem{TMDB_ID: "100", LibraryTitle: "Movies", Edition: "4K"}, PosterSets: []models.DBPosterSetDetail{set("a")}},
			{MediaItem: models.MediaItem{TMDB_ID: "100", LibraryTitle: "Movies"}, PosterSets: []models.DBPosterSetDetail{set("b"), set("a")}},
		}}, logging.LogErrorInfo{}
	}
	getQueueJobs = func(_ context.Context) ([]database.DownloadQueueJob, logging.LogErrorInfo) {
		return jobs, logging.LogErrorInfo{}
	}
	addToQueue = func(_ context.Context, item models.DBSavedItem) logging.LogErrorInfo {
		queued = append(queued, item)
		return logging.LogErrorInfo{}
	}
	t.Cleanup(func() {
		getHistoryEntry, getSavedSets, getQueueJobs, addToQueue = database.GetDownloadHistoryEntry, database.GetAllSavedSets, database.GetAllDownloadQueueJobs, AddToQueue
	})

	// Only the failed set is queued, for the entry's edition
	if title, Err := RetryFailedDownload(ctx, 1); Err.Message != "" || title != "Hope" {
		t.Fatalf("retry = %q %q", title, Err.Message)
	}
	if len(queued) != 1 || queued[0].MediaItem.Edition != "" || len(queued[0].PosterSets) != 1 || queued[0].PosterSets[0].ID != "a" {
		t.Errorf("queued = %+v", queued)
	}

	// Nothing is queued without a set, for a set that isn't saved, for a missing entry, or while the item is in the queue
	for _, id := range []int64{2, 3, 4} {
		if _, Err := RetryFailedDownload(ctx, id); Err.Message == "" {
			t.Errorf("retry %d should fail", id)
		}
	}
	jobs = []database.DownloadQueueJob{{ID: 9, TMDB_ID: "100", LibraryTitle: "Movies", Status: "pending"}}
	if _, Err := RetryFailedDownload(ctx, 1); Err.Message == "" {
		t.Error("retry while queued should fail")
	}
	if len(queued) != 1 {
		t.Errorf("queued %d times, want 1", len(queued))
	}
}
