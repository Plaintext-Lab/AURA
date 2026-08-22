package models

// ArtworkSuggestion represents a media item with at least one AURA-managed artwork coverage gap.
type ArtworkSuggestion struct {
	// Media identity
	TMDB_ID      string `json:"tmdb_id"`
	LibraryTitle string `json:"library_title"`
	Edition      string `json:"edition"`
	RatingKey    string `json:"rating_key"`
	Type         string `json:"type"` // "movie" or "show"
	Title        string `json:"title"`
	Year         int    `json:"year"`

	// Coverage gap reason codes and human-readable explanation.
	// Possible reason codes:
	//   "no_set"                  – no saved artwork set exists
	//   "no_poster"               – no poster is selected across all saved sets
	//   "no_backdrop"             – no backdrop is selected across all saved sets
	//   "missing_season_posters"  – one or more current seasons lack a season poster in the saved set
	//   "missing_title_cards"     – one or more current episodes lack a title card in the saved set
	ReasonCodes []string `json:"reason_codes"`
	ReasonText  string   `json:"reason_text"`

	// Coverage counts
	PosterCount       int `json:"poster_count"`        // number of saved sets with a poster selected
	BackdropCount     int `json:"backdrop_count"`      // number of saved sets with a backdrop selected
	SeasonPosterTotal int `json:"season_poster_total"` // total seasons in the current library
	SeasonPosterCount int `json:"season_poster_count"` // seasons covered by saved set images
	TitleCardTotal    int `json:"title_card_total"`    // total episodes in the current library
	TitleCardCount    int `json:"title_card_count"`    // episodes covered by saved set images

	// Availability
	HasSavedSet   bool `json:"has_saved_set"`   // at least one saved set exists
	HasMediuxSets bool `json:"has_mediux_sets"` // MediUX sets are available (from cached data)
}

// ArtworkSuggestionsFilter controls pagination, filtering and sorting for the suggestions endpoint.
type ArtworkSuggestionsFilter struct {
	// Filters
	LibraryTitles  []string `json:"library_titles"`   // restrict to specific library titles
	MediaType      string   `json:"media_type"`       // "" | "movie" | "show"
	GapType        string   `json:"gap_type"`         // "" | "no_set" | "no_poster" | "no_backdrop" | "missing_season_posters" | "missing_title_cards"
	HasMediuxSets  string   `json:"has_mediux_sets"`  // "" | "true" | "false"
	SearchTitle    string   `json:"search_title"`     // partial title match
	SearchTMDB_ID  string   `json:"search_tmdb_id"`   // exact TMDB ID match

	// Pagination
	ItemsPerPage int `json:"items_per_page"` // default 25, -1 for all
	PageNumber   int `json:"page_number"`    // 1-based

	// Sorting
	SortOption string `json:"sort_option"` // "title" | "year" | "library"
	SortOrder  string `json:"sort_order"`  // "asc" | "desc"
}
