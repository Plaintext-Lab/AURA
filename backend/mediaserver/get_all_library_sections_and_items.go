package mediaserver

import (
	"aura/cache"
	"aura/config"
	"aura/logging"
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"
)

var (
	warmupMu   sync.Mutex
	warmupDone bool
)

func GetAllLibrarySectionsAndItems(ctx context.Context, force bool) (success bool) {
	// If we already did a run that satisfies this request, skip.
	warmupMu.Lock()
	alreadyDone := warmupDone
	warmupMu.Unlock()
	if alreadyDone && !force {
		return true
	}

	success = getAllLibrarySectionsAndItemsImpl(ctx)
	if success {
		warmupMu.Lock()
		warmupDone = true
		warmupMu.Unlock()
	}

	return success
}

func getAllLibrarySectionsAndItemsImpl(ctx context.Context) (success bool) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Fetching All Library Sections and Items", logging.LevelDebug)
	defer logAction.Complete()

	// Main media server
	success = loadLibrariesIntoCache(ctx, config.Current.MediaServer, cache.LibraryStore, true)
	if !success {
		return false
	}
	cache.CollectionsStore.LastFullUpdate = time.Now().Unix()

	// Other media servers load in the background, so a slow or unreachable one doesn't hold up aura
	go RefreshOtherMediaServers()
	return true
}

// loadLibrariesIntoCache loads the libraries and items of a media server into a library cache.
// Collections are only loaded for the main media server.
func loadLibrariesIntoCache(ctx context.Context, server config.Config_MediaServer, store *cache.MediaServerLibraryCache, withCollections bool) (success bool) {
	ctx, logAction := logging.AddSubActionToContext(ctx, fmt.Sprintf("Loading Libraries: %s", server.Name), logging.LevelDebug)
	defer logAction.Complete()

	msClient, Err := NewMediaServerClient(&server)
	if Err.Message != "" {
		logAction.SetErrorFromInfo(Err)
		return false
	}

	configuredSections := slices.Clone(server.Libraries)

	// Sort sections by Title to ensure consistent order
	sort.SliceStable(configuredSections, func(i, j int) bool {
		return configuredSections[i].Title < configuredSections[j].Title
	})

	logAction.AppendResult("num_sections", len(configuredSections))

	ejRanCollections := false

	for _, section := range configuredSections {
		found, Err := msClient.GetLibrarySectionDetails(ctx, &section)
		if Err.Message != "" || !found {
			continue
		}

		// Update the collections cache for this section
		if withCollections && (section.Type == "movie" || section.Type == "mixed") && !ejRanCollections {
			GetMovieCollections(ctx, section)
			if server.Type == "Emby" || server.Type == "Jellyfin" {
				ejRanCollections = true
			}
		}

		pageSize := 1000
		start := 0
		expectedTotal := 0

		for {

			items, totalSize, Err := msClient.GetLibrarySectionItems(ctx, section, strconv.Itoa(start), strconv.Itoa(pageSize))
			if Err.Message != "" {
				return false
			}
			logging.LOGGER.Info().Timestamp().
				Str("media_server", server.Name).
				Str("section_title", section.Title).
				Str("section_id", section.ID).
				Int("fetched_items", len(items)).
				Int("start_index", start).
				Int("total_size", totalSize).
				Msg("Fetched library section items")

			if totalSize > 0 {
				expectedTotal = totalSize
			}
			if len(items) == 0 {
				break
			}

			sectionForCache := section
			sectionForCache.TotalSize = expectedTotal
			sectionForCache.MediaItems = items

			// Update Library Cache
			store.UpdateSection(&sectionForCache)

			start += len(items)

			if expectedTotal > 0 && start >= expectedTotal {
				break
			}

		}

	}
	store.LastFullUpdate = time.Now().Unix()
	return true
}
