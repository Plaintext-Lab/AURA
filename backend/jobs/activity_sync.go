package jobs

import (
	"aura/activity"
	"aura/config"
	"aura/database"
	"aura/logging"
	"context"
	"fmt"
	"time"
)

// RunActivitySync executes one activity sync pass.
// It is called both by the cron scheduler and the manual "Sync now" action.
func RunActivitySync() {
	cfg := config.Current.ActivitySource
	if !cfg.Enabled {
		return
	}

	ctx, ld := logging.CreateLoggingContext(context.Background(), "Activity Sync")
	defer ld.Log()
	logAction := ld.AddAction("Syncing activity from "+cfg.Provider, logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)
	defer logAction.Complete()

	provider, err := activity.NewProvider(cfg.Provider)
	if err != nil {
		activity.DefaultCache.SetError(err)
		logAction.SetError("Unknown activity provider", err.Error(), map[string]any{"provider": cfg.Provider})
		return
	}

	windowDays := cfg.ActivityWindowDays
	if windowDays <= 0 {
		windowDays = 30
	}

	sourceCfg := activity.ActivitySourceConfig{
		Provider:         cfg.Provider,
		BaseURL:          cfg.BaseURL,
		ApiToken:         cfg.ApiToken,
		TracearrServerID: cfg.TracearrServerID,
	}

	req := activity.ActivitySyncRequest{
		Source:     sourceCfg,
		WindowDays: windowDays,
		MaxPages:   50,
	}

	summaries, syncErr := provider.Sync(ctx, req)
	if syncErr != nil {
		activity.DefaultCache.SetError(syncErr)
		logAction.SetError("Activity sync failed", syncErr.Error(), map[string]any{"provider": cfg.Provider})
		return
	}

	// Persist to database
	rows := make([]database.ActivitySummaryRow, 0, len(summaries))
	for _, s := range summaries {
		rows = append(rows, database.ActivitySummaryRow{
			Source:         s.Source,
			LibraryID:      s.LibraryID,
			RatingKey:      s.RatingKey,
			GrandparentKey: s.GrandparentKey,
			MediaType:      string(s.MediaType),
			TmdbID:         s.TmdbID,
			TvdbID:         s.TvdbID,
			Title:          s.Title,
			PlayCount:      s.PlayCount,
			WatchTimeSecs:  s.WatchTimeSeconds,
			LastWatched:    s.LastWatched.UTC().Format(time.RFC3339),
			WindowStart:    s.WindowStart.UTC().Format(time.RFC3339),
			WindowEnd:      s.WindowEnd.UTC().Format(time.RFC3339),
		})
	}

	if dbErr := database.UpsertActivitySummaries(ctx, rows); dbErr.Message != "" {
		activity.DefaultCache.SetError(fmt.Errorf("db upsert: %s", dbErr.Message))
		logAction.SetError("Failed to persist activity summaries", dbErr.Message, nil)
		return
	}

	// Update in-memory cache
	activity.DefaultCache.Update(summaries)

	logging.LOGGER.Info().Timestamp().
		Str("provider", cfg.Provider).
		Int("summaries", len(summaries)).
		Msg("Activity sync completed")
}
