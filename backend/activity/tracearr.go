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

// TracearrAdapter implements ActivityProvider for Tracearr.
type TracearrAdapter struct{}

// ---------- Tracearr API response shapes ----------

type tracearrStatusResponse struct {
	Version string `json:"version"`
}

type tracearrHistoryRecord struct {
	ServerID       string `json:"server_id"`
	LibraryID      string `json:"library_id"`
	RatingKey      string `json:"rating_key"`
	GrandparentKey string `json:"grandparent_key"`
	ShowKey        string `json:"show_key"`
	MediaType      string `json:"media_type"` // "movie" | "episode"
	TmdbID         string `json:"tmdb_id"`
	TvdbID         string `json:"tvdb_id"`
	ShowTmdbID     string `json:"show_tmdb_id"`
	Title          string `json:"title"`
	ShowTitle      string `json:"show_title"`
	DurationMs     int64  `json:"duration_ms"`
	Watched        bool   `json:"watched"`
	WatchedAt      string `json:"watched_at"`
}

type tracearrHistoryPage struct {
	Records    []tracearrHistoryRecord `json:"records"`
	NextCursor string                  `json:"next_cursor"`
}

// ---------- TestConnection ----------

func (a *TracearrAdapter) TestConnection(ctx context.Context, cfg ActivitySourceConfig) (ProviderInfo, error) {
	endpoint, err := tracearrURL(cfg.BaseURL, "status", nil)
	if err != nil {
		return ProviderInfo{}, fmt.Errorf("tracearr: build URL: %w", err)
	}

	headers := httpx.MakeAuthHeader("Authorization", cfg.ApiToken)
	_, body, reqErr := httpx.MakeHTTPRequest(ctx, endpoint, "GET", headers, 15, nil, "Tracearr")
	if reqErr.Message != "" {
		return ProviderInfo{}, fmt.Errorf("tracearr: %s", reqErr.Message)
	}

	var resp tracearrStatusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return ProviderInfo{}, fmt.Errorf("tracearr: decode status: %w", err)
	}

	return ProviderInfo{
		Name:     "Tracearr",
		Version:  resp.Version,
		ServerID: cfg.TracearrServerID,
	}, nil
}

// ---------- Sync ----------

func (a *TracearrAdapter) Sync(ctx context.Context, req ActivitySyncRequest) ([]MediaActivitySummary, error) {
	windowEnd := time.Now().UTC()
	windowStart := windowEnd.AddDate(0, 0, -req.WindowDays)

	aggregate := map[string]*MediaActivitySummary{}

	maxPages := req.MaxPages
	if maxPages <= 0 {
		maxPages = 50
	}

	cursor := ""
	headers := httpx.MakeAuthHeader("Authorization", req.Source.ApiToken)

	for page := 0; page < maxPages; page++ {
		params := map[string]string{
			"after":  windowStart.Format(time.RFC3339),
			"before": windowEnd.Format(time.RFC3339),
		}
		if req.Source.TracearrServerID != "" {
			params["server_id"] = req.Source.TracearrServerID
		}
		if cursor != "" {
			params["cursor"] = cursor
		}

		endpoint, err := tracearrURL(req.Source.BaseURL, "public/history", params)
		if err != nil {
			return nil, fmt.Errorf("tracearr: build URL page %d: %w", page, err)
		}

		_, body, reqErr := httpx.MakeHTTPRequest(ctx, endpoint, "GET", headers, 30, nil, "Tracearr")
		if reqErr.Message != "" {
			return nil, fmt.Errorf("tracearr: page %d: %s", page, reqErr.Message)
		}

		var pg tracearrHistoryPage
		if err := json.Unmarshal(body, &pg); err != nil {
			return nil, fmt.Errorf("tracearr: decode page %d: %w", page, err)
		}

		if len(pg.Records) == 0 {
			break
		}

		for _, rec := range pg.Records {
			// Privacy: no user/device/player/IP fields in public API response.
			tracearrAggregateRecord(aggregate, rec, req.Source.Provider, windowStart, windowEnd)
		}

		if pg.NextCursor == "" {
			break
		}
		cursor = pg.NextCursor
	}

	return flattenSummaries(aggregate), nil
}

// ---------- helpers ----------

func tracearrAggregateRecord(agg map[string]*MediaActivitySummary, rec tracearrHistoryRecord, source string, windowStart, windowEnd time.Time) {
	mapKey := rec.RatingKey
	ratingKey := rec.RatingKey
	grandparentKey := rec.GrandparentKey
	mediaType := MediaType(rec.MediaType)
	title := rec.Title
	tmdbID := rec.TmdbID
	tvdbID := rec.TvdbID

	if rec.MediaType == "episode" {
		showKey := rec.ShowKey
		if showKey == "" {
			showKey = rec.GrandparentKey
		}
		if showKey != "" {
			mapKey = "show-" + showKey
			ratingKey = showKey
			grandparentKey = showKey
			mediaType = MediaTypeShow
			title = rec.ShowTitle
			if rec.ShowTmdbID != "" {
				tmdbID = rec.ShowTmdbID
			}
			tvdbID = ""
		}
	}

	watchedAt := time.Now().UTC()
	if rec.WatchedAt != "" {
		if t, err := time.Parse(time.RFC3339, rec.WatchedAt); err == nil {
			watchedAt = t.UTC()
		}
	}

	watchSeconds := rec.DurationMs / 1000
	if !rec.Watched {
		watchSeconds = 0
	}

	if existing, ok := agg[mapKey]; ok {
		existing.PlayCount++
		existing.WatchTimeSeconds += watchSeconds
		if watchedAt.After(existing.LastWatched) {
			existing.LastWatched = watchedAt
		}
	} else {
		agg[mapKey] = &MediaActivitySummary{
			Source:           source,
			LibraryID:        rec.LibraryID,
			RatingKey:        ratingKey,
			GrandparentKey:   grandparentKey,
			MediaType:        mediaType,
			TmdbID:           tmdbID,
			TvdbID:           tvdbID,
			Title:            title,
			PlayCount:        1,
			WatchTimeSeconds: watchSeconds,
			LastWatched:      watchedAt,
			WindowStart:      windowStart,
			WindowEnd:        windowEnd,
		}
	}
}

func tracearrURL(baseURL, apiPath string, params map[string]string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "api", "v2", apiPath)
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}
