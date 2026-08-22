/** An individual artwork coverage suggestion returned by GET /api/suggestions/artwork */
export interface ArtworkSuggestion {
  /** TMDB ID of the media item */
  tmdb_id: string;
  /** Library title this item belongs to */
  library_title: string;
  /** Edition string (empty for standard releases) */
  edition: string;
  /** Internal media-server rating key */
  rating_key: string;
  /** "movie" | "show" */
  type: "movie" | "show";
  /** Display title */
  title: string;
  /** Release year */
  year: number;

  /**
   * Machine-readable reason codes explaining the coverage gap.
   * Possible values:
   *   "no_set"                 – no saved artwork set exists
   *   "no_poster"              – no poster is selected across all saved sets
   *   "no_backdrop"            – no backdrop is selected across all saved sets
   *   "missing_season_posters" – one or more current seasons lack a season poster
   *   "missing_title_cards"    – one or more current episodes lack a title card
   */
  reason_codes: string[];
  /** Human-readable explanation of the coverage gap */
  reason_text: string;

  /** Number of saved sets that have a poster selected */
  poster_count: number;
  /** Number of saved sets that have a backdrop selected */
  backdrop_count: number;
  /** Total seasons in the current library (shows only) */
  season_poster_total: number;
  /** Seasons covered by saved set images (shows only) */
  season_poster_count: number;
  /** Total episodes in the current library (shows only) */
  title_card_total: number;
  /** Episodes covered by saved set images (shows only) */
  title_card_count: number;

  /** Whether at least one saved set exists for this item */
  has_saved_set: boolean;
  /** Whether MediUX sets are available (from cached data) */
  has_mediux_sets: boolean;
}

/** Response envelope from GET /api/suggestions/artwork */
export interface ArtworkSuggestionsResponse {
  items: ArtworkSuggestion[];
  total_items: number;
}

/** Query parameters for the artwork suggestions endpoint */
export interface ArtworkSuggestionsParams {
  library_titles?: string;
  media_type?: string;
  gap_type?: string;
  has_mediux_sets?: string;
  search_title?: string;
  search_tmdb_id?: string;
  items_per_page?: number;
  page_number?: number;
  sort_option?: string;
  sort_order?: "asc" | "desc";
}
