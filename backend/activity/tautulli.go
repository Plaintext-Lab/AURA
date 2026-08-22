package activity

import (
	"aura/utils/httpx"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"time"
)

// TautulliAdapter implements ActivityProvider for Tautulli.
type TautulliAdapter struct{}

// ---------- Tautulli API response shapes ----------

type tautulliSystemInfoResponse struct {
	Response struct {
		Result string `json:"result"`
		Data   struct {
			TautulliVersion string `json:"tautulli_version"`
		} `json:"data"`
	} `json:"response"`
}

type tautulliHistoryRow struct {
	SectionID            int    `json:"section_id"`
	RatingKey            int    `json:"rating_key"`
	GrandparentRatingKey int    `json:"grandparent_rating_key"`
	MediaType            string `json:"media_type"` // "movie" | "episode"
	GrandparentTitle     string `json:"grandparent_title"`
	Title                string `json:"title"`
	GrandparentGUID      string `json:"grandparent_guid"`
	GUID                 string `json:"guid"`
	Date                 int64  `json:"date"`
	PlayDuration         int    `json:"play_duration"`
}

type tautulliHistoryResponse struct {
	Response struct {
		Result string `json:"result"`
		Data   struct {
			RecordsFiltered int                  `json:"recordsFiltered"`
			Data            []tautulliHistoryRow `json:"data"`
		} `json:"data"`
	} `json:"response"`
}

// ---------- TestConnection ----------

func (a *TautulliAdapter) TestConnection(ctx context.Context, cfg ActivitySourceConfig) (ProviderInfo, error) {
	endpoint, err := tautulliURL(cfg.BaseURL, map[string]string{
		"apikey": cfg.ApiToken,
		"cmd":    "get_tautulli_info",
	})
	if err != nil {
		return ProviderInfo{}, fmt.Errorf("tautulli: build URL: %w", err)
	}

	_, body, reqErr := httpx.MakeHTTPRequest(ctx, endpoint, "GET", nil, 15, nil, "Tautulli")
	if reqErr.Message != "" {
		return ProviderInfo{}, fmt.Errorf("tautulli: %s", reqErr.Message)
	}

	var resp tautulliSystemInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ProviderInfo{}, fmt.Errorf("tautulli: decode info response: %w", err)
	}
	if resp.Response.Result != "success" {
		return ProviderInfo{}, fmt.Errorf("tautulli: API returned non-success result")
	}

	return ProviderInfo{
		Name:    "Tautulli",
		Version: resp.Response.Data.TautulliVersion,
	}, nil
}

// ---------- Sync ----------

func (a *TautulliAdapter) Sync(ctx context.Context, req ActivitySyncRequest) ([]MediaActivitySummary, error) {
	windowEnd := time.Now().UTC()
	windowStart := windowEnd.AddDate(0, 0, -req.WindowDays)

	aggregate := map[string]*MediaActivitySummary{}

	pageSize := 100
	start := 0
	maxPages := req.MaxPages
	if maxPages <= 0 {
		maxPages = 50
	}

	for page := 0; page < maxPages; page++ {
		endpoint, err := tautulliURL(req.Source.BaseURL, map[string]string{
			"apikey":     req.Source.ApiToken,
			"cmd":        "get_history",
			"after":      fmt.Sprintf("%d", windowStart.Unix()),
			"before":     fmt.Sprintf("%d", windowEnd.Unix()),
			"media_type": "movie,episode",
			"length":     fmt.Sprintf("%d", pageSize),
			"start":      fmt.Sprintf("%d", start),
		})
		if err != nil {
			return nil, fmt.Errorf("tautulli: build URL page %d: %w", page, err)
		}

		_, body, reqErr := httpx.MakeHTTPRequest(ctx, endpoint, "GET", nil, 30, nil, "Tautulli")
		if reqErr.Message != "" {
			return nil, fmt.Errorf("tautulli: page %d request: %s", page, reqErr.Message)
		}

		var hist tautulliHistoryResponse
		if err := json.Unmarshal(body, &hist); err != nil {
			return nil, fmt.Errorf("tautulli: decode history page %d: %w", page, err)
		}
		if hist.Response.Result != "success" {
			return nil, fmt.Errorf("tautulli: API non-success on page %d", page)
		}

		rows := hist.Response.Data.Data
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			// Privacy: only aggregate play counts, durations and timestamps.
			// User, device, player, IP and location fields are never captured.
			tautulliAggregateRow(aggregate, row, req.Source.Provider, windowStart, windowEnd)
		}

		start += len(rows)
		if start >= hist.Response.Data.RecordsFiltered {
			break
		}
	}

	return flattenSummaries(aggregate), nil
}

// ---------- helpers ----------

func tautulliAggregateRow(agg map[string]*MediaActivitySummary, row tautulliHistoryRow, source string, windowStart, windowEnd time.Time) {
	mapKey := fmt.Sprintf("%d", row.RatingKey)
	mediaType := MediaType(row.MediaType)
	ratingKeyStr := fmt.Sprintf("%d", row.RatingKey)
	grandparentKeyStr := ""
	sectionIDStr := fmt.Sprintf("%d", row.SectionID)
	titleForSummary := row.Title

	if row.MediaType == "episode" && row.GrandparentRatingKey != 0 {
		mapKey = fmt.Sprintf("show-%d", row.GrandparentRatingKey)
		ratingKeyStr = fmt.Sprintf("%d", row.GrandparentRatingKey)
		grandparentKeyStr = ratingKeyStr
		mediaType = MediaTypeShow
		titleForSummary = row.GrandparentTitle
	}

	tmdbID, tvdbID := tautulliExtractIDs(row.GrandparentGUID, row.GUID, row.MediaType)
	played := time.Unix(row.Date, 0).UTC()

	if existing, ok := agg[mapKey]; ok {
		existing.PlayCount++
		existing.WatchTimeSeconds += int64(row.PlayDuration)
		if played.After(existing.LastWatched) {
			existing.LastWatched = played
		}
	} else {
		agg[mapKey] = &MediaActivitySummary{
			Source:           source,
			LibraryID:        sectionIDStr,
			RatingKey:        ratingKeyStr,
			GrandparentKey:   grandparentKeyStr,
			MediaType:        mediaType,
			TmdbID:           tmdbID,
			TvdbID:           tvdbID,
			Title:            titleForSummary,
			PlayCount:        1,
			WatchTimeSeconds: int64(row.PlayDuration),
			LastWatched:      played,
			WindowStart:      windowStart,
			WindowEnd:        windowEnd,
		}
	}
}

// tautulliExtractIDs parses "tmdb://NNN" or "tvdb://NNN" from GUID strings.
func tautulliExtractIDs(grandparentGUID, guid, mediaType string) (tmdbID, tvdbID string) {
	for _, g := range []string{grandparentGUID, guid} {
		if len(g) > 7 && g[:7] == "tmdb://" {
			tmdbID = g[7:]
		}
		if len(g) > 7 && g[:7] == "tvdb://" {
			tvdbID = g[7:]
		}
	}
	return
}

func tautulliURL(baseURL string, params map[string]string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "api", "v2")
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
