package database

import (
	"aura/cache"
	"aura/config"
	"aura/logging"
	"aura/models"
	"context"
	"database/sql"
)

const LATEST_DB_VERSION = 8

var Client DB

type SQliteDB struct {
	conn *sql.DB
}

type DB interface {
	// Open Database Connection
	GetDBConnection(ctx context.Context) (conn *sql.DB, newDB bool, Err logging.LogErrorInfo)

	// Close Database Connection
	Close(ctx context.Context) (Err logging.LogErrorInfo)

	// Initialize the database connection
	Init(ctx context.Context) (newDB bool, Err logging.LogErrorInfo)

	// Create Main Tables
	CreateTables(ctx context.Context) (Err logging.LogErrorInfo)

	// Create VERSION table
	CreateVersionTable(ctx context.Context) (Err logging.LogErrorInfo)

	// Get all stored config values
	GetAllConfigValues(ctx context.Context) (values map[string]string, Err logging.LogErrorInfo)

	// Save config values. Only changed values are written. If replace is true, other stored values are removed.
	SaveConfigValues(ctx context.Context, values map[string]string, replace bool) (Err logging.LogErrorInfo)

	// Get the current database version (from VERSION table)
	GetCurrentVersion(ctx context.Context) (version int, Err logging.LogErrorInfo)

	// Update the VERSION table to newVersion
	UpdateVersionTable(ctx context.Context, newVersion int) (Err logging.LogErrorInfo)

	// Perform database vacuuming to optimize the database
	Vacuum(ctx context.Context) (Err logging.LogErrorInfo)

	// Create Auth table
	CreateAuthTable(ctx context.Context) (Err logging.LogErrorInfo)

	// Get Auth Token Secret
	GetAuthTokenSecret(ctx context.Context) (secret string, Err logging.LogErrorInfo)

	// Backup Database
	Backup(ctx context.Context, currentVersion, newVersion int) (Err logging.LogErrorInfo)

	// Upsert Converted Saved Item
	UpsertSavedItem(ctx context.Context, newItem models.DBSavedItem) (Err logging.LogErrorInfo)

	// Check Media Item Exists
	CheckIfMediaItemExists(ctx context.Context, TMDB_ID, libraryTitle, edition string) (ignored bool, ignoredMode string, sets []models.DBSavedSet, logErr logging.LogErrorInfo)

	// Get All Media Items
	GetAllMediaItems(ctx context.Context) (items []models.MediaItem, logErr logging.LogErrorInfo)

	// Get All Media Items with info on whether they have saved sets or are ignored
	GetAllMediaItemsWithFlags(ctx context.Context) (items []MediaItemWithFlags, logErr logging.LogErrorInfo)

	// Update Media Item
	UpdateMediaItem(ctx context.Context, updatedItem models.MediaItem) (Err logging.LogErrorInfo)

	// Delete Media Item and Ignored Item entries for a given TMDB ID and Library Title
	DeleteMediaItemAndIgnoredStatus(ctx context.Context, TMDB_ID, libraryTitle, edition string) (Err logging.LogErrorInfo)

	// Get All Saved Sets
	GetAllSavedSets(ctx context.Context, dbFilter models.DBFilter) (out PagedSavedItems, logErr logging.LogErrorInfo)

	// Get All Unique Users from Saved Sets
	GetAllUniqueUsers(ctx context.Context) (users []string, logErr logging.LogErrorInfo)

	// Get Count of Saved Sets (Unique TMDB ID + Library Title combinations)
	//GetCountSavedSets(ctx context.Context) (count int, logErr logging.LogErrorInfo)

	// Delete Poster Set (and associated images) by ID
	DeletePosterSetForMediaItem(ctx context.Context, tmdbID, libraryTitle, edition, setID string) (Err logging.LogErrorInfo)

	// Delete All Poster Sets for Media Item
	DeleteAllPosterSetsForMediaItem(ctx context.Context, tmdbID, libraryTitle, edition string) (Err logging.LogErrorInfo)

	// Ignore Media Item
	IgnoreMediaItem(ctx context.Context, tmdbID, libraryTitle, edition, mode, currentSets string) (Err logging.LogErrorInfo)

	// Stop Ignoring Media Item
	StopIgnoringMediaItem(ctx context.Context, TMDB_ID, libraryTitle, edition string) (Err logging.LogErrorInfo)

	// Get Temp Ignored Items
	GetTempIgnoredItems(ctx context.Context) (items []models.MediaItem, Err logging.LogErrorInfo)

	// Update Media Item on_server flag
	UpdateMediaItemOnServer(ctx context.Context, tmdbID string, libraryTitle string, edition string, onServer bool) (logErr logging.LogErrorInfo)

	// Reconcile a Media Item whose Edition changed
	ReconcileMediaItemEdition(ctx context.Context, tmdbID, libraryTitle, oldEdition string, updatedItem models.MediaItem) (Err logging.LogErrorInfo)

	// Enqueue a new Download Queue Job
	EnqueueDownloadQueueJob(ctx context.Context, item models.DBSavedItem) (jobID int64, Err logging.LogErrorInfo)

	// Atomically claim the next pending Download Queue Job, if any
	ClaimNextDownloadQueueJob(ctx context.Context, workerID string) (job DownloadQueueJob, found bool, Err logging.LogErrorInfo)

	// Mark a Download Queue Job as finished with its result
	FinishDownloadQueueJob(ctx context.Context, jobID int64, status string, message string, errs, warnings []string) (Err logging.LogErrorInfo)

	// Get All Download Queue Jobs
	GetAllDownloadQueueJobs(ctx context.Context) (jobs []DownloadQueueJob, Err logging.LogErrorInfo)

	// Delete a single Download Queue Job by ID
	DeleteDownloadQueueJob(ctx context.Context, jobID int64) (Err logging.LogErrorInfo)

	// Delete all Download Queue Jobs (except those currently processing)
	DeleteAllDownloadQueueJobs(ctx context.Context) (Err logging.LogErrorInfo)

	// Insert a Download History entry, applying retention cleanup
	InsertDownloadHistoryEntry(ctx context.Context, entry DownloadHistoryEntry) (Err logging.LogErrorInfo)

	// Get Download History entries
	GetDownloadHistory(ctx context.Context, limit, offset int) (entries []DownloadHistoryEntry, total int, Err logging.LogErrorInfo)

	// Delete a single Download History entry by ID
	DeleteDownloadHistoryEntry(ctx context.Context, id int64) (Err logging.LogErrorInfo)

	// Delete all Download History entries
	DeleteAllDownloadHistory(ctx context.Context) (Err logging.LogErrorInfo)

	// Get all saved Silo Item IDs
	GetAllSiloItemIDs(ctx context.Context) (items []SiloItemIDs, Err logging.LogErrorInfo)

	// Insert or update Silo Item IDs
	UpsertSiloItemIDs(ctx context.Context, items []SiloItemIDs) (Err logging.LogErrorInfo)
}

func NewDatabaseClient() (DB, logging.LogErrorInfo) {
	return &SQliteDB{}, logging.LogErrorInfo{}
}

func GetDBConnection(ctx context.Context) (conn *sql.DB, newDB bool, Err logging.LogErrorInfo) {
	if Client == nil {
		return nil, false, logging.Error_DBClientNotInitialized()
	}
	return Client.GetDBConnection(ctx)
}

func Init(ctx context.Context) (newDB bool, Err logging.LogErrorInfo) {
	dbClient, err := NewDatabaseClient()
	if err.Message != "" {
		return false, err
	}
	Client = dbClient
	return dbClient.Init(ctx)
}

func Close(ctx context.Context) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.LogErrorInfo{}
	}
	return Client.Close(ctx)
}

func GetAllConfigValues(ctx context.Context) (values map[string]string, Err logging.LogErrorInfo) {
	if Client == nil {
		return map[string]string{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetAllConfigValues(ctx)
}

func SaveConfigValues(ctx context.Context, values map[string]string, replace bool) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.SaveConfigValues(ctx, values, replace)
}

// configStore stores the config in the database
type configStore struct{}

func (configStore) GetAllConfigValues(ctx context.Context) (map[string]string, logging.LogErrorInfo) {
	return GetAllConfigValues(ctx)
}

func (configStore) SaveConfigValues(ctx context.Context, values map[string]string, replace bool) logging.LogErrorInfo {
	return SaveConfigValues(ctx, values, replace)
}

// ConfigStore returns the store used to load and save the config
func ConfigStore() config.Store {
	return configStore{}
}

func CreateVersionTable(ctx context.Context) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.CreateVersionTable(ctx)
}

func GetCurrentVersion(ctx context.Context) (version int, Err logging.LogErrorInfo) {
	if Client == nil {
		return 0, logging.Error_DBClientNotInitialized()
	}
	return Client.GetCurrentVersion(ctx)
}

func UpdateVersionTable(ctx context.Context, newVersion int) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.UpdateVersionTable(ctx, newVersion)
}

func CreateAuthTable(ctx context.Context) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.CreateAuthTable(ctx)
}

func GetAuthTokenSecret(ctx context.Context) (secret string, Err logging.LogErrorInfo) {
	if Client == nil {
		return "", logging.Error_DBClientNotInitialized()
	}
	return Client.GetAuthTokenSecret(ctx)
}

func Vacuum(ctx context.Context) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.Vacuum(ctx)
}

func CreateTables(ctx context.Context) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.CreateTables(ctx)
}

func Backup(ctx context.Context, currentVersion, newVersion int) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.Backup(ctx, currentVersion, newVersion)
}

func UpsertSavedItem(ctx context.Context, newItem models.DBSavedItem) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	Err = Client.UpsertSavedItem(ctx, newItem)
	if Err.Message != "" {
		return Err
	}

	// Silo uses one item for every library it is in, so images applied in one library are shown in all of them
	// Save the same sets for the item in the other libraries as well
	if config.Current.MediaServer.Type == "Silo" && newItem.MediaItem.RatingKey != "" {
		for _, otherItem := range cache.LibraryStore.GetMediaItemsByRatingKey(newItem.MediaItem.RatingKey) {
			if otherItem.LibraryTitle == newItem.MediaItem.LibraryTitle {
				continue
			}
			sharedItem := newItem
			sharedItem.MediaItem.LibraryTitle = otherItem.LibraryTitle
			// Use the file info for the other library, if it is known
			if otherItem.Movie != nil {
				sharedItem.MediaItem.Movie = otherItem.Movie
			}
			if otherItem.Series != nil {
				sharedItem.MediaItem.Series = otherItem.Series
			}
			Err = Client.UpsertSavedItem(ctx, sharedItem)
			if Err.Message != "" {
				return Err
			}
		}
	}
	return logging.LogErrorInfo{}
}

func CheckIfMediaItemExists(ctx context.Context, TMDB_ID, libraryTitle, edition string) (ignored bool, ignoreMode string, sets []models.DBSavedSet, logErr logging.LogErrorInfo) {
	if Client == nil {
		return false, "", []models.DBSavedSet{}, logging.Error_DBClientNotInitialized()
	}
	return Client.CheckIfMediaItemExists(ctx, TMDB_ID, libraryTitle, edition)
}

func GetAllMediaItems(ctx context.Context) (items []models.MediaItem, logErr logging.LogErrorInfo) {
	if Client == nil {
		return []models.MediaItem{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetAllMediaItems(ctx)
}

func GetAllMediaItemsWithFlags(ctx context.Context) (items []MediaItemWithFlags, logErr logging.LogErrorInfo) {
	if Client == nil {
		return []MediaItemWithFlags{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetAllMediaItemsWithFlags(ctx)
}

func UpdateMediaItem(ctx context.Context, updatedItem models.MediaItem) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.UpdateMediaItem(ctx, updatedItem)
}

func DeleteMediaItemAndIgnoredStatus(ctx context.Context, TMDB_ID, libraryTitle, edition string) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.DeleteMediaItemAndIgnoredStatus(ctx, TMDB_ID, libraryTitle, edition)
}

func GetAllSavedSets(ctx context.Context, dbFilter models.DBFilter) (out PagedSavedItems, logErr logging.LogErrorInfo) {
	if Client == nil {
		return PagedSavedItems{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetAllSavedSets(ctx, dbFilter)
}

func GetAllUniqueUsers(ctx context.Context) (users []string, logErr logging.LogErrorInfo) {
	if Client == nil {
		return []string{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetAllUniqueUsers(ctx)
}

// func GetCountSavedSets(ctx context.Context) (count int, logErr logging.LogErrorInfo) {
// 	if Client == nil {
// 		return 0, logging.Error_DBClientNotInitialized()
// 	}
// 	return Client.GetCountSavedSets(ctx)
// }

func DeletePosterSetForMediaItem(ctx context.Context, tmdbID, libraryTitle, edition, setID string) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	Err = Client.DeletePosterSetForMediaItem(ctx, tmdbID, libraryTitle, edition, setID)
	if Err.Message != "" {
		return Err
	}

	// Silo uses one item for every library it is in, so remove the set from the other libraries as well
	for _, otherLibraryTitle := range getSiloSharedLibraryTitles(tmdbID, libraryTitle, edition) {
		logSharedLibraryError(Client.DeletePosterSetForMediaItem(ctx, tmdbID, otherLibraryTitle, edition, setID), tmdbID, otherLibraryTitle)
	}
	return logging.LogErrorInfo{}
}

func DeleteAllPosterSetsForMediaItem(ctx context.Context, tmdbID, libraryTitle, edition string) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	Err = Client.DeleteAllPosterSetsForMediaItem(ctx, tmdbID, libraryTitle, edition)
	if Err.Message != "" {
		return Err
	}

	// Silo uses one item for every library it is in, so remove the sets from the other libraries as well
	for _, otherLibraryTitle := range getSiloSharedLibraryTitles(tmdbID, libraryTitle, edition) {
		logSharedLibraryError(Client.DeleteAllPosterSetsForMediaItem(ctx, tmdbID, otherLibraryTitle, edition), tmdbID, otherLibraryTitle)
	}
	return logging.LogErrorInfo{}
}

func IgnoreMediaItem(ctx context.Context, tmdbID, libraryTitle, edition, mode, currentSets string) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	Err = Client.IgnoreMediaItem(ctx, tmdbID, libraryTitle, edition, mode, currentSets)
	if Err.Message != "" {
		return Err
	}

	// Silo uses one item for every library it is in, so ignore the item in the other libraries as well
	for _, otherLibraryTitle := range getSiloSharedLibraryTitles(tmdbID, libraryTitle, edition) {
		logSharedLibraryError(Client.IgnoreMediaItem(ctx, tmdbID, otherLibraryTitle, edition, mode, currentSets), tmdbID, otherLibraryTitle)
	}
	return logging.LogErrorInfo{}
}

func StopIgnoringMediaItem(ctx context.Context, TMDB_ID, libraryTitle, edition string) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	Err = Client.StopIgnoringMediaItem(ctx, TMDB_ID, libraryTitle, edition)
	if Err.Message != "" {
		return Err
	}

	// Silo uses one item for every library it is in, so stop ignoring the item in the other libraries as well
	for _, otherLibraryTitle := range getSiloSharedLibraryTitles(TMDB_ID, libraryTitle, edition) {
		logSharedLibraryError(Client.StopIgnoringMediaItem(ctx, TMDB_ID, otherLibraryTitle, edition), TMDB_ID, otherLibraryTitle)
	}
	return logging.LogErrorInfo{}
}

func GetTempIgnoredItems(ctx context.Context) (items []models.MediaItem, Err logging.LogErrorInfo) {
	if Client == nil {
		return []models.MediaItem{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetTempIgnoredItems(ctx)
}

func UpdateMediaItemOnServer(ctx context.Context, tmdbID string, libraryTitle string, edition string, onServer bool) (logErr logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.UpdateMediaItemOnServer(ctx, tmdbID, libraryTitle, edition, onServer)
}

func ReconcileMediaItemEdition(ctx context.Context, tmdbID, libraryTitle, oldEdition string, updatedItem models.MediaItem) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.ReconcileMediaItemEdition(ctx, tmdbID, libraryTitle, oldEdition, updatedItem)
}

func EnqueueDownloadQueueJob(ctx context.Context, item models.DBSavedItem) (jobID int64, Err logging.LogErrorInfo) {
	if Client == nil {
		return 0, logging.Error_DBClientNotInitialized()
	}
	return Client.EnqueueDownloadQueueJob(ctx, item)
}

func ClaimNextDownloadQueueJob(ctx context.Context, workerID string) (job DownloadQueueJob, found bool, Err logging.LogErrorInfo) {
	if Client == nil {
		return DownloadQueueJob{}, false, logging.Error_DBClientNotInitialized()
	}
	return Client.ClaimNextDownloadQueueJob(ctx, workerID)
}

func FinishDownloadQueueJob(ctx context.Context, jobID int64, status string, message string, errs, warnings []string) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.FinishDownloadQueueJob(ctx, jobID, status, message, errs, warnings)
}

func GetAllDownloadQueueJobs(ctx context.Context) (jobs []DownloadQueueJob, Err logging.LogErrorInfo) {
	if Client == nil {
		return []DownloadQueueJob{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetAllDownloadQueueJobs(ctx)
}

func DeleteDownloadQueueJob(ctx context.Context, jobID int64) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.DeleteDownloadQueueJob(ctx, jobID)
}

func DeleteAllDownloadQueueJobs(ctx context.Context) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.DeleteAllDownloadQueueJobs(ctx)
}

func InsertDownloadHistoryEntry(ctx context.Context, entry DownloadHistoryEntry) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.InsertDownloadHistoryEntry(ctx, entry)
}

func GetDownloadHistory(ctx context.Context, limit, offset int) (entries []DownloadHistoryEntry, total int, Err logging.LogErrorInfo) {
	if Client == nil {
		return []DownloadHistoryEntry{}, 0, logging.Error_DBClientNotInitialized()
	}
	return Client.GetDownloadHistory(ctx, limit, offset)
}

func DeleteDownloadHistoryEntry(ctx context.Context, id int64) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.DeleteDownloadHistoryEntry(ctx, id)
}

func DeleteAllDownloadHistory(ctx context.Context) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.DeleteAllDownloadHistory(ctx)
}

func GetAllSiloItemIDs(ctx context.Context) (items []SiloItemIDs, Err logging.LogErrorInfo) {
	if Client == nil {
		return []SiloItemIDs{}, logging.Error_DBClientNotInitialized()
	}
	return Client.GetAllSiloItemIDs(ctx)
}

func UpsertSiloItemIDs(ctx context.Context, items []SiloItemIDs) (Err logging.LogErrorInfo) {
	if Client == nil {
		return logging.Error_DBClientNotInitialized()
	}
	return Client.UpsertSiloItemIDs(ctx, items)
}

// getSiloSharedLibraryTitles returns the other libraries that have the same item.
// Silo uses one item (rating key) for every library it is in, so changes in one library apply to all of them.
// Returns nothing for other media servers.
func getSiloSharedLibraryTitles(tmdbID, libraryTitle, edition string) []string {
	if config.Current.MediaServer.Type != "Silo" {
		return nil
	}
	item, found := cache.LibraryStore.GetMediaItemFromSectionByTMDBIDAndEdition(libraryTitle, tmdbID, edition)
	if !found || item.RatingKey == "" {
		return nil
	}
	libraryTitles := []string{}
	for _, otherItem := range cache.LibraryStore.GetMediaItemsByRatingKey(item.RatingKey) {
		if otherItem.LibraryTitle != libraryTitle {
			libraryTitles = append(libraryTitles, otherItem.LibraryTitle)
		}
	}
	return libraryTitles
}

// logSharedLibraryError logs a failed change in another library without failing the original change
func logSharedLibraryError(Err logging.LogErrorInfo, tmdbID, libraryTitle string) {
	if Err.Message == "" {
		return
	}
	logging.LOGGER.Warn().Timestamp().Str("tmdb_id", tmdbID).Str("library_title", libraryTitle).Str("error", Err.Message).
		Msg("Failed to apply the change to the same Silo item in another library")
}
