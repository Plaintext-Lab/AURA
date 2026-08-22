package database

import (
	"aura/logging"
	"aura/models"
	"context"
	"fmt"
	"strings"
)

// PagedArtworkSuggestions is the paginated result set returned by GetArtworkSuggestions.
type PagedArtworkSuggestions struct {
	Items []models.ArtworkSuggestion
	Total int
}

// GetArtworkSuggestions returns media items that have at least one AURA-managed artwork
// coverage gap (no saved set, missing poster/backdrop/season-poster/title-card).
//
// The query never contacts Tautulli, Tracearr, MediUX, or the media server – it operates
// entirely from local SQLite data.
func (s *SQliteDB) GetArtworkSuggestions(ctx context.Context, f models.ArtworkSuggestionsFilter) (out PagedArtworkSuggestions, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Getting Artwork Suggestions from Database", logging.LevelInfo)
	defer logAction.Complete()

	out.Items = make([]models.ArtworkSuggestion, 0)

	if s == nil || s.conn == nil {
		logAction.SetError("Database connection is nil", "", map[string]any{})
		return out, *logAction.Error
	}

	// ---------------------------------------------------------------
	// Build WHERE clause for the outer query
	// ---------------------------------------------------------------
	var whereParts []string
	var args []any

	// Only on-server items
	whereParts = append(whereParts, "mi.on_server = 1")

	// Not ignored
	whereParts = append(whereParts, `NOT EXISTS (
		SELECT 1 FROM IgnoredItems ii
		WHERE ii.tmdb_id = mi.tmdb_id
		  AND ii.library_title = mi.library_title
		  AND ii.edition = mi.edition
	)`)

	// Library filter
	if len(f.LibraryTitles) > 0 {
		placeholders := make([]string, len(f.LibraryTitles))
		for i, lt := range f.LibraryTitles {
			placeholders[i] = "?"
			args = append(args, lt)
		}
		whereParts = append(whereParts, fmt.Sprintf("mi.library_title IN (%s)", strings.Join(placeholders, ",")))
	}

	// Media type filter
	if f.MediaType == "movie" || f.MediaType == "show" {
		whereParts = append(whereParts, "mi.type = ?")
		args = append(args, f.MediaType)
	}

	// Title search
	if f.SearchTitle != "" {
		whereParts = append(whereParts, "mi.title LIKE ?")
		args = append(args, "%"+f.SearchTitle+"%")
	}

	// TMDB ID search
	if f.SearchTMDB_ID != "" {
		whereParts = append(whereParts, "mi.tmdb_id = ?")
		args = append(args, f.SearchTMDB_ID)
	}

	// Coverage gap type filter
	// These are applied after the coverage CTEs, so we embed them in HAVING-style checks
	// via gapFilter below.
	gapFilter := buildGapFilter(f.GapType)

	// has_mediux_sets filter – handled in Go from the in-memory cache after the DB query
	_ = f.HasMediuxSets

	whereSQL := ""
	if len(whereParts) > 0 {
		whereSQL = "WHERE " + strings.Join(whereParts, "\n  AND ")
	}

	// ---------------------------------------------------------------
	// Sorting
	// ---------------------------------------------------------------
	sortCol := "mi.title"
	switch strings.ToLower(strings.TrimSpace(f.SortOption)) {
	case "year":
		sortCol = "mi.year"
	case "library":
		sortCol = "mi.library_title"
	default:
		sortCol = "mi.title"
	}

	sortDir := "ASC"
	if strings.EqualFold(f.SortOrder, "desc") {
		sortDir = "DESC"
	}

	// ---------------------------------------------------------------
	// Pagination
	// ---------------------------------------------------------------
	pageItems := f.ItemsPerPage
	if pageItems < 0 {
		pageItems = -1 // SQLite LIMIT -1 = no limit
	}
	if pageItems == 0 {
		pageItems = 25
	}
	if pageItems > 250 {
		pageItems = 250
	}
	pageNumber := f.PageNumber
	if pageNumber <= 0 {
		pageNumber = 1
	}
	limitOffset := fmt.Sprintf("LIMIT %d OFFSET %d", pageItems, (pageNumber-1)*pageItems)
	if pageItems == -1 {
		limitOffset = ""
	}

	// ---------------------------------------------------------------
	// Core CTE query
	// ---------------------------------------------------------------
	// We join coverage CTEs so we can identify gaps in a single pass.
	coreSQL := fmt.Sprintf(`
WITH
-- Best saved-item flags per media item (aggregate across all saved sets)
best_saved AS (
  SELECT
    tmdb_id,
    library_title,
    edition,
    COUNT(DISTINCT poster_set_id)          AS set_count,
    MAX(poster_selected)                    AS poster_selected,
    MAX(backdrop_selected)                  AS backdrop_selected,
    MAX(season_poster_selected)             AS season_poster_selected,
    MAX(special_season_poster_selected)     AS special_season_poster_selected,
    MAX(titlecard_selected)                 AS titlecard_selected
  FROM SavedItems
  GROUP BY tmdb_id, library_title, edition
),
-- Season poster image coverage: distinct season numbers covered by any saved set's images
season_poster_cov AS (
  SELECT
    si.tmdb_id,
    si.library_title,
    si.edition,
    COUNT(DISTINCT imgf.image_season_number) AS covered_seasons
  FROM SavedItems si
  JOIN ImageFiles imgf
    ON imgf.poster_set_id = si.poster_set_id
   AND imgf.item_tmdb_id  = si.tmdb_id
   AND imgf.image_type    = 'season_poster'
  GROUP BY si.tmdb_id, si.library_title, si.edition
),
-- Title card image coverage: distinct (season, episode) pairs covered by any saved set's images
titlecard_cov AS (
  SELECT
    si.tmdb_id,
    si.library_title,
    si.edition,
    COUNT(DISTINCT imgf.image_season_number || '_' || imgf.image_episode_number) AS covered_episodes
  FROM SavedItems si
  JOIN ImageFiles imgf
    ON imgf.poster_set_id = si.poster_set_id
   AND imgf.item_tmdb_id  = si.tmdb_id
   AND imgf.image_type    = 'titlecard'
  GROUP BY si.tmdb_id, si.library_title, si.edition
),
-- Season counts per media item
season_counts AS (
  SELECT
    mi.tmdb_id,
    mi.library_title,
    mi.edition,
    COUNT(s.id) AS total_seasons
  FROM MediaItems mi
  JOIN Series sr  ON sr.media_item_id = mi.id
  JOIN Seasons s  ON s.series_id      = sr.id
  GROUP BY mi.tmdb_id, mi.library_title, mi.edition
),
-- Episode counts per media item
episode_counts AS (
  SELECT
    mi.tmdb_id,
    mi.library_title,
    mi.edition,
    COUNT(e.id) AS total_episodes
  FROM MediaItems mi
  JOIN Series sr  ON sr.media_item_id = mi.id
  JOIN Seasons s  ON s.series_id      = sr.id
  JOIN Episodes e ON e.season_id      = s.id
  GROUP BY mi.tmdb_id, mi.library_title, mi.edition
),
-- Main coverage view
coverage AS (
  SELECT
    mi.tmdb_id,
    mi.library_title,
    mi.edition,
    mi.rating_key,
    mi.type,
    mi.title,
    mi.year,
    COALESCE(bs.set_count, 0)              AS set_count,
    COALESCE(bs.poster_selected, 0)        AS poster_selected,
    COALESCE(bs.backdrop_selected, 0)      AS backdrop_selected,
    COALESCE(bs.season_poster_selected, 0) AS season_poster_selected,
    COALESCE(bs.titlecard_selected, 0)     AS titlecard_selected,
    COALESCE(sc.total_seasons, 0)          AS total_seasons,
    COALESCE(spc.covered_seasons, 0)       AS covered_seasons,
    COALESCE(ec.total_episodes, 0)         AS total_episodes,
    COALESCE(tcc.covered_episodes, 0)      AS covered_episodes
  FROM MediaItems mi
  LEFT JOIN best_saved      bs  ON bs.tmdb_id  = mi.tmdb_id  AND bs.library_title  = mi.library_title  AND bs.edition  = mi.edition
  LEFT JOIN season_counts   sc  ON sc.tmdb_id  = mi.tmdb_id  AND sc.library_title  = mi.library_title  AND sc.edition  = mi.edition
  LEFT JOIN season_poster_cov spc ON spc.tmdb_id = mi.tmdb_id AND spc.library_title = mi.library_title AND spc.edition = mi.edition
  LEFT JOIN episode_counts  ec  ON ec.tmdb_id  = mi.tmdb_id  AND ec.library_title  = mi.library_title  AND ec.edition  = mi.edition
  LEFT JOIN titlecard_cov   tcc ON tcc.tmdb_id = mi.tmdb_id  AND tcc.library_title = mi.library_title  AND tcc.edition = mi.edition
  %s
)
`, whereSQL)

	// ---------------------------------------------------------------
	// Gap predicate (applied once WHERE is already in the CTE above;
	// re-applied on the outer SELECT so we filter to items with gaps)
	// ---------------------------------------------------------------
	gapPredicate := buildGapPredicate(gapFilter)

	// ---------------------------------------------------------------
	// Count query
	// ---------------------------------------------------------------
	countSQL := coreSQL + fmt.Sprintf(`
SELECT COUNT(*)
FROM coverage mi
WHERE %s;
`, gapPredicate)

	if err := s.conn.QueryRowContext(ctx, countSQL, args...).Scan(&out.Total); err != nil {
		logAction.SetError("Failed to count artwork suggestions", "", map[string]any{"error": err.Error()})
		return out, *logAction.Error
	}

	// ---------------------------------------------------------------
	// Data query
	// ---------------------------------------------------------------
	dataSQL := coreSQL + fmt.Sprintf(`
SELECT
  mi.tmdb_id,
  mi.library_title,
  mi.edition,
  mi.rating_key,
  mi.type,
  mi.title,
  mi.year,
  mi.set_count,
  mi.poster_selected,
  mi.backdrop_selected,
  mi.total_seasons,
  mi.covered_seasons,
  mi.total_episodes,
  mi.covered_episodes
FROM coverage mi
WHERE %s
ORDER BY %s %s
%s;
`, gapPredicate, sortCol, sortDir, limitOffset)

	rows, err := s.conn.QueryContext(ctx, dataSQL, args...)
	if err != nil {
		logAction.SetError("Failed to query artwork suggestions", "", map[string]any{"error": err.Error()})
		return out, *logAction.Error
	}
	defer rows.Close()

	for rows.Next() {
		var (
			item        models.ArtworkSuggestion
			setCount    int
			posterSel   int
			backdropSel int
		)
		if err := rows.Scan(
			&item.TMDB_ID,
			&item.LibraryTitle,
			&item.Edition,
			&item.RatingKey,
			&item.Type,
			&item.Title,
			&item.Year,
			&setCount,
			&posterSel,
			&backdropSel,
			&item.SeasonPosterTotal,
			&item.SeasonPosterCount,
			&item.TitleCardTotal,
			&item.TitleCardCount,
		); err != nil {
			logAction.SetError("Failed to scan artwork suggestion row", "", map[string]any{"error": err.Error()})
			return out, *logAction.Error
		}

		item.HasSavedSet = setCount > 0
		item.HasMediuxSets = false // populated from cache in the HTTP handler
		item.PosterCount = posterSel
		item.BackdropCount = backdropSel

		// Build reason codes
		item.ReasonCodes, item.ReasonText = buildReasonCodes(item)
		out.Items = append(out.Items, item)
	}

	if err := rows.Err(); err != nil {
		logAction.SetError("Row iteration error for artwork suggestions", "", map[string]any{"error": err.Error()})
		return out, *logAction.Error
	}

	return out, Err
}

// buildGapFilter normalises the caller's gap_type filter into an internal token.
func buildGapFilter(gapType string) string {
	switch strings.ToLower(strings.TrimSpace(gapType)) {
	case "no_set":
		return "no_set"
	case "no_poster":
		return "no_poster"
	case "no_backdrop":
		return "no_backdrop"
	case "missing_season_posters":
		return "missing_season_posters"
	case "missing_title_cards":
		return "missing_title_cards"
	default:
		return "any"
	}
}

// buildGapPredicate returns the SQL predicate that filters coverage rows to those with gaps.
// The CTE alias for the coverage table is "mi".
func buildGapPredicate(gapFilter string) string {
	switch gapFilter {
	case "no_set":
		return "mi.set_count = 0"
	case "no_poster":
		return "(mi.set_count = 0 OR mi.poster_selected = 0)"
	case "no_backdrop":
		return "(mi.set_count = 0 OR mi.backdrop_selected = 0)"
	case "missing_season_posters":
		return `(mi.set_count = 0 OR (mi.type = 'show' AND mi.total_seasons > 0 AND mi.covered_seasons < mi.total_seasons))`
	case "missing_title_cards":
		return `(mi.set_count = 0 OR (mi.type = 'show' AND mi.total_episodes > 0 AND mi.covered_episodes < mi.total_episodes))`
	default:
		// "any" – any coverage gap
		return `(
      mi.set_count = 0
      OR mi.poster_selected   = 0
      OR mi.backdrop_selected = 0
      OR (mi.type = 'show' AND mi.total_seasons  > 0 AND mi.covered_seasons  < mi.total_seasons)
      OR (mi.type = 'show' AND mi.total_episodes > 0 AND mi.covered_episodes < mi.total_episodes)
    )`
	}
}

// buildReasonCodes derives human-readable reason codes from a populated ArtworkSuggestion.
func buildReasonCodes(item models.ArtworkSuggestion) (codes []string, text string) {
	if !item.HasSavedSet {
		codes = append(codes, "no_set")
	}
	if item.PosterCount == 0 {
		codes = append(codes, "no_poster")
	}
	if item.BackdropCount == 0 {
		codes = append(codes, "no_backdrop")
	}
	if item.Type == "show" && item.SeasonPosterTotal > 0 && item.SeasonPosterCount < item.SeasonPosterTotal {
		codes = append(codes, "missing_season_posters")
	}
	if item.Type == "show" && item.TitleCardTotal > 0 && item.TitleCardCount < item.TitleCardTotal {
		codes = append(codes, "missing_title_cards")
	}

	// Human-readable summary
	parts := make([]string, 0, len(codes))
	for _, c := range codes {
		switch c {
		case "no_set":
			parts = append(parts, "no saved artwork set")
		case "no_poster":
			parts = append(parts, "no poster selected")
		case "no_backdrop":
			parts = append(parts, "no backdrop selected")
		case "missing_season_posters":
			parts = append(parts, fmt.Sprintf(
				"season posters missing for %d of %d season(s)",
				item.SeasonPosterTotal-item.SeasonPosterCount,
				item.SeasonPosterTotal,
			))
		case "missing_title_cards":
			parts = append(parts, fmt.Sprintf(
				"title cards missing for %d of %d episode(s)",
				item.TitleCardTotal-item.TitleCardCount,
				item.TitleCardTotal,
			))
		}
	}

	if len(parts) == 0 {
		text = "no coverage gaps detected"
	} else {
		text = strings.Join(parts, "; ")
	}
	return codes, text
}
