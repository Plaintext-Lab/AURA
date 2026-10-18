package downloadqueue

import (
	"aura/database"
	"aura/logging"
	"aura/models"
	"context"
	"fmt"
)

// Set in tests
var (
	getHistoryEntry = database.GetDownloadHistoryEntry
	getSavedSets    = database.GetAllSavedSets
	getQueueJobs    = database.GetAllDownloadQueueJobs
	addToQueue      = AddToQueue
)

// RetryFailedDownload adds a failed download's set back to the Download Queue, from the item's saved set
func RetryFailedDownload(ctx context.Context, historyID int64) (title string, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf("Retry Failed Download %d", historyID), logging.LevelInfo)
	defer logAction.Complete()

	entry, found, Err := getHistoryEntry(ctx, historyID)
	if Err.Message != "" {
		return "", Err
	}
	if !found {
		logAction.SetError("This failed download no longer exists", "Refresh the page", map[string]any{"id": historyID})
		return "", *logAction.Error
	}
	title = entry.MediaItemTitle

	// A failure from before a set was downloaded has nothing to retry
	if entry.SetID == "" {
		logAction.SetError(fmt.Sprintf("%s has no set to retry", title), "Download a set from the item's page", map[string]any{"id": historyID})
		return title, *logAction.Error
	}

	jobs, Err := getQueueJobs(ctx)
	if Err.Message != "" {
		return title, Err
	}
	for _, job := range jobs {
		if job.TMDB_ID == entry.TMDB_ID && job.LibraryTitle == entry.LibraryTitle && job.Edition == entry.Edition &&
			(job.Status == "pending" || job.Status == "processing") {
			logAction.SetError(fmt.Sprintf("%s is already in the Download Queue", title), "Wait for it to finish", map[string]any{"job_id": job.ID})
			return title, *logAction.Error
		}
	}

	saved, Err := getSavedSets(ctx, models.DBFilter{
		ItemTMDB_ID:      entry.TMDB_ID,
		ItemLibraryTitle: entry.LibraryTitle,
		SetID:            entry.SetID,
		ItemsPerPage:     -1,
	})
	if Err.Message != "" {
		return title, Err
	}
	for _, item := range saved.Items {
		if item.MediaItem.Edition != entry.Edition {
			continue
		}
		for _, set := range item.PosterSets {
			if set.ID != entry.SetID {
				continue
			}
			// Only the failed set, the item's other saved sets downloaded fine
			Err = addToQueue(ctx, models.DBSavedItem{MediaItem: item.MediaItem, PosterSets: []models.DBPosterSetDetail{set}})
			if Err.Message != "" {
				return title, Err
			}
			logAction.AppendResult("queued", fmt.Sprintf("%s set %s", title, set.ID))
			return title, logging.LogErrorInfo{}
		}
	}

	logAction.SetError(fmt.Sprintf("Set %s isn't saved for %s", entry.SetID, title), "Download the set again from the item's page", map[string]any{"id": historyID})
	return title, *logAction.Error
}
