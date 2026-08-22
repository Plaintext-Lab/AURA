package routes_db

import (
	"aura/cache"
	"aura/config"
	"aura/database"
	downloadqueue "aura/download/queue"
	"aura/logging"
	"aura/mediaserver"
	"aura/mediux"
	"aura/models"
	"context"
	"fmt"
)

// PropagateToLinkedLibraries checks whether the saved item's library belongs to a
// linked-library group and, if so, creates/updates saved items for every other
// library in the group that contains the same media (matched by TMDB ID + edition
// for movies, TMDB ID only for shows).
//
// This function runs asynchronously; failures are logged but do not affect the
// caller.  A failure in one target never rolls back successful targets.
func PropagateToLinkedLibraries(ctx context.Context, savedItem models.DBSavedItem, fullSet models.DBPosterSetDetail) {
	ctx, ld := logging.CreateLoggingContext(ctx, "Propagate To Linked Libraries")
	logAction := ld.AddAction("Propagate artwork to linked libraries", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)
	defer ld.Log()

	sourceLibraryTitle := savedItem.MediaItem.LibraryTitle
	tmdbID := savedItem.MediaItem.TMDB_ID
	edition := savedItem.MediaItem.Edition
	mediaType := savedItem.MediaItem.Type

	// Resolve the stable library ID for the source library
	sourceLibraryID := libraryIDForTitle(sourceLibraryTitle)
	if sourceLibraryID == "" {
		logAction.AppendResult("skipped", "source library has no stable ID")
		logAction.Complete()
		return
	}

	// Find all groups that contain this library
	groups, Err := database.GetGroupsForLibrary(ctx, sourceLibraryID)
	if Err.Message != "" {
		logAction.SetError("Failed to query linked groups", Err.Message, nil)
		return
	}
	if len(groups) == 0 {
		logAction.Complete()
		return
	}

	for _, group := range groups {
		if group.MediaType != mediaType {
			continue
		}

		// Upsert the group-level policy so it is always current
		existingPolicy, _, _ := database.GetLibraryGroupPolicyByTMDB(ctx, group.ID, tmdbID, edition)
		policyID := existingPolicy.ID
		if policyID == "" {
			policyID = newRandomID()
		}
		policy := models.LibraryGroupPolicy{
			ID:            policyID,
			GroupID:       group.ID,
			TMDB_ID:       tmdbID,
			Edition:       edition,
			SetID:         fullSet.ID,
			SelectedTypes: fullSet.SelectedTypes,
			AutoDownload:  fullSet.AutoDownload,
		}
		if Err := database.UpsertLibraryGroupPolicy(ctx, policy); Err.Message != "" {
			logAction.AppendResult(fmt.Sprintf("group_%s_policy_error", group.ID), Err.Message)
		}

		// Apply the set to every other library in the group
		for _, targetLibraryID := range group.LibraryIDs {
			if targetLibraryID == sourceLibraryID {
				continue
			}
			targetLibraryTitle := libraryTitleForID(targetLibraryID)
			if targetLibraryTitle == "" {
				logAction.AppendResult(fmt.Sprintf("library_%s", targetLibraryID), "library not found in config")
				continue
			}

			// Look up the matching media item in the target library
			var targetItem *models.MediaItem
			if mediaType == "show" {
				item, found := cache.LibraryStore.GetMediaItemFromSectionByTMDBID(targetLibraryTitle, tmdbID)
				if !found || item == nil {
					logAction.AppendResult(fmt.Sprintf("library_%s_match", targetLibraryID), "item not found in target library")
					continue
				}
				targetItem = item
			} else {
				item, found := cache.LibraryStore.GetMediaItemFromSectionByTMDBIDAndEdition(targetLibraryTitle, tmdbID, edition)
				if !found || item == nil {
					logAction.AppendResult(fmt.Sprintf("library_%s_match", targetLibraryID), "item not found in target library (edition)")
					continue
				}
				targetItem = item
			}

			// Fetch full item details from the media server
			if found, detailErr := mediaserver.GetMediaItemDetails(ctx, targetItem); detailErr.Message != "" || !found {
				logAction.AppendResult(fmt.Sprintf("library_%s_details", targetLibraryID), "failed to get media item details")
				continue
			}

			// Fetch the full poster set if images are missing (same set ID)
			targetSet := fullSet
			if len(targetSet.Images) == 0 {
				switch fullSet.Type {
				case "show":
					showSet, _, sErr := mediux.GetShowSetByID(ctx, fullSet.ID, targetLibraryTitle, edition)
					if sErr.Message == "" {
						targetSet.PosterSet = showSet.PosterSet
					}
				case "movie":
					movieSet, _, sErr := mediux.GetMovieSetByID(ctx, fullSet.ID, targetLibraryTitle, edition)
					if sErr.Message == "" {
						targetSet.PosterSet = movieSet.PosterSet
					}
				}
			}

			targetSaveItem := models.DBSavedItem{
				MediaItem:  *targetItem,
				PosterSets: []models.DBPosterSetDetail{targetSet},
			}

			if Err := database.UpsertSavedItem(ctx, targetSaveItem); Err.Message != "" {
				logAction.AppendResult(fmt.Sprintf("library_%s_save", targetLibraryID), Err.Message)
				continue
			}

			// Update the cache
			_, _, dbSets, _ := database.CheckIfMediaItemExists(ctx, tmdbID, targetLibraryTitle, edition)
			targetItem.DBSavedSets = dbSets
			cache.LibraryStore.UpdateMediaItem(targetLibraryTitle, targetItem)

			// Queue the download for this target so images are actually applied to the media server.
			// A queue failure is logged as a partial failure but does not roll back the DB save.
			if qErr := downloadqueue.AddToQueue(ctx, targetSaveItem); qErr.Message != "" {
				logAction.AppendResult(fmt.Sprintf("library_%s_queue", targetLibraryID), "partial failure: queued failed: "+qErr.Message)
			}

			logAction.AppendResult(fmt.Sprintf("library_%s", targetLibraryID), "propagated")
		}
	}

	logAction.Complete()
}

// libraryIDForTitle returns the stable library ID for a given library title from config.
func libraryIDForTitle(title string) string {
	for _, lib := range config.Current.MediaServer.Libraries {
		if lib.Title == title {
			return lib.ID
		}
	}
	return ""
}

// libraryTitleForID returns the library display title for a given stable library ID from config.
func libraryTitleForID(id string) string {
	for _, lib := range config.Current.MediaServer.Libraries {
		if lib.ID == id {
			return lib.Title
		}
	}
	return ""
}
