package routes_suggestions

import (
	"aura/cache"
	"aura/database"
	"aura/logging"
	"aura/models"
	"aura/utils/httpx"
	"net/http"
	"strconv"
	"strings"
)

type artworkSuggestionsResponse struct {
	Items      []models.ArtworkSuggestion `json:"items"`
	TotalItems int                        `json:"total_items"`
}

// GetArtworkSuggestions godoc
// @Summary      Get Artwork Suggestions
// @Description  Returns paginated media items that have at least one AURA-managed artwork coverage gap (missing set, poster, backdrop, season poster, or title card). Results can be filtered and sorted. No external provider requests are made per page.
// @Tags         Suggestions
// @Produce      json
// @Param        library_titles  query  string  false  "Filter by library titles (comma-separated)"
// @Param        media_type      query  string  false  "Filter by media type: movie or show"
// @Param        gap_type        query  string  false  "Filter by gap type: no_set, no_poster, no_backdrop, missing_season_posters, missing_title_cards, or any (default)"
// @Param        has_mediux_sets query  string  false  "Filter by MediUX set availability: true or false"
// @Param        search_title    query  string  false  "Partial title search"
// @Param        search_tmdb_id  query  string  false  "Exact TMDB ID match"
// @Param        items_per_page  query  int     false  "Items per page (default 25, -1 for all)"
// @Param        page_number     query  int     false  "Page number, 1-based (default 1)"
// @Param        sort_option     query  string  false  "Sort by: title (default), year, or library"
// @Param        sort_order      query  string  false  "Sort direction: asc (default) or desc"
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse "Unauthorized (only when Auth.Enabled=true)"
// @Success      200  {object}  httpx.JSONResponse{data=artworkSuggestionsResponse}
// @Failure      500  {object}  httpx.JSONResponse "Internal Server Error"
// @Router       /api/suggestions/artwork [get]
func GetArtworkSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Get Artwork Suggestions", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	var f models.ArtworkSuggestionsFilter

	// Library filter
	if lt := r.URL.Query().Get("library_titles"); lt != "" {
		f.LibraryTitles = strings.Split(lt, ",")
	}

	// Media type
	f.MediaType = r.URL.Query().Get("media_type")

	// Gap type
	f.GapType = r.URL.Query().Get("gap_type")

	// MediUX availability (applied in Go after DB query)
	f.HasMediuxSets = r.URL.Query().Get("has_mediux_sets")

	// Title / TMDB ID search
	f.SearchTitle = r.URL.Query().Get("search_title")
	f.SearchTMDB_ID = r.URL.Query().Get("search_tmdb_id")

	// Pagination
	f.ItemsPerPage = 25
	if v := r.URL.Query().Get("items_per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.ItemsPerPage = n
		}
	}
	f.PageNumber = 1
	if v := r.URL.Query().Get("page_number"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.PageNumber = n
		}
	}

	// Sorting
	f.SortOption = r.URL.Query().Get("sort_option")
	f.SortOrder = r.URL.Query().Get("sort_order")

	result, dbErr := database.GetArtworkSuggestions(ctx, f)
	if dbErr.Message != "" {
		logAction.SetErrorFromInfo(dbErr)
		httpx.SendResponse(w, ld, artworkSuggestionsResponse{})
		return
	}

	// Populate has_mediux_sets from the in-memory cache (no external requests).
	for i := range result.Items {
		result.Items[i].HasMediuxSets = cache.MediuxItems.CheckItemExists(
			result.Items[i].Type, result.Items[i].TMDB_ID,
		)
	}

	// Apply has_mediux_sets filter in Go (cache-based, cannot be done in SQL).
	// Update total to reflect the post-filter count so pagination is accurate.
	items := result.Items
	totalItems := result.Total
	if f.HasMediuxSets == "true" {
		filtered := make([]models.ArtworkSuggestion, 0)
		for _, it := range items {
			if it.HasMediuxSets {
				filtered = append(filtered, it)
			}
		}
		items = filtered
		totalItems = len(items)
	} else if f.HasMediuxSets == "false" {
		filtered := make([]models.ArtworkSuggestion, 0)
		for _, it := range items {
			if !it.HasMediuxSets {
				filtered = append(filtered, it)
			}
		}
		items = filtered
		totalItems = len(items)
	}

	httpx.SendResponse(w, ld, artworkSuggestionsResponse{
		Items:      items,
		TotalItems: totalItems,
	})
}
