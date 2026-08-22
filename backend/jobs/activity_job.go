package jobs

import (
	"aura/config"
	"aura/logging"
	"runtime/debug"

	"github.com/robfig/cron/v3"
)

// StartActivitySyncJob registers the activity sync cron job based on the
// current configuration. It is safe to call multiple times; an existing job
// is removed before re-registering.
func StartActivitySyncJob() error {
	mu.Lock()
	defer mu.Unlock()

	if c == nil {
		logging.LOGGER.Error().Timestamp().Msg("Cron Jobs Scheduler is not initialized")
		return nil
	}

	if activitySyncJobID != 0 {
		c.Remove(activitySyncJobID)
		activitySyncJobID = 0
	}

	if !config.Current.ActivitySource.Enabled {
		logging.LOGGER.Info().Timestamp().Msg("Activity Sync Job: disabled (activity source not enabled)")
		return nil
	}

	spec := config.Current.ActivitySource.RefreshInterval
	if spec == "" {
		spec = "*/30 * * * *"
	}

	var err error
	var capturedID cron.EntryID
	activitySyncJobID, err = c.AddFunc(spec, func() {
		defer func() {
			if r := recover(); r != nil {
				logging.LOGGER.Error().
					Timestamp().
					Interface("recover", r).
					Str("stack", string(debug.Stack())).
					Msg("PANIC: in scheduled Activity Sync Job")
			}
		}()
		RunActivitySync()
		// Use capturedID (set right after AddFunc returns) to avoid reading
		// the package-level activitySyncJobID without the mutex.
		logging.LOGGER.Info().Timestamp().
			Str("next_run", c.Entry(capturedID).Next.String()).
			Msg("Activity Sync Job Completed")
	})
	if err != nil {
		return err
	}
	capturedID = activitySyncJobID
	jobSpecs[activitySyncJobID] = spec

	logging.LOGGER.Info().Timestamp().
		Str("cron", spec).
		Str("provider", config.Current.ActivitySource.Provider).
		Msg("Activity Sync Job Started")
	return nil
}
