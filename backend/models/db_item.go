package models

import "time"

// What is used to save a record into the database
// This contains the MediaItem details, as well as an array of PosterSets that are associated with it
type DBSavedItem struct {
	MediaItem  MediaItem           `json:"media_item"`
	PosterSets []DBPosterSetDetail `json:"poster_sets,omitempty"`
	Source     string              `json:"source,omitempty"` // What added it to the Download Queue, empty when it was the user
}

// The sources of Download Queue items that the user didn't add
const (
	SourceFavoriteCreatorsNewItem  = "favorite_creators_new_item"  // A new item, from a favorite creator
	SourceFavoriteCreatorsAllItems = "favorite_creators_all_items" // Run for All Items, from a favorite creator
	SourceSiloPlugin               = "silo_plugin"                 // Silo used an image from the aura plugin
)

// PosterSetDetail groups poster set details per media item.
type DBPosterSetDetail struct {
	PosterSet
	LastDownloaded            time.Time     `json:"last_downloaded"`
	SelectedTypes             SelectedTypes `json:"selected_types"`
	AutoDownload              bool          `json:"auto_download"`
	AutoAddNewCollectionItems bool          `json:"auto_add_new_collection_items"`
	ToDelete                  bool          `json:"to_delete"` // Flag to indicate if the poster set should be deleted (Not used in DB)
}

type PosterSet struct {
	BaseSetInfo
	Images []ImageFile `json:"images"`
}

type DBFilter struct {
	ItemTMDB_ID       string   `json:"item_tmdb_id"`
	ItemLibraryTitle  string   `json:"item_library_title"`
	ItemYear          int      `json:"item_year"`
	ItemTitle         string   `json:"item_title"`
	SetID             string   `json:"set_id"`
	LibraryTitles     []string `json:"library_titles"`
	ImageTypes        []string `json:"image_types"`
	Autodownload      string   `json:"autodownload"`
	MultiSetOnly      bool     `json:"multiset_only"`
	Usernames         []string `json:"usernames"`
	MediaItemOnServer string   `json:"media_item_on_server"`
	ItemsPerPage      int      `json:"items_per_page"`
	PageNumber        int      `json:"page_number"`
	SortOption        string   `json:"sort_option"`
	SortOrder         string   `json:"sort_order"`
}
