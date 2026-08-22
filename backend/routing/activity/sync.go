package routes_activity

import (
	"aura/config"
	"aura/jobs"
	"aura/logging"
	"aura/utils/httpx"
	"net/http"
)

type syncNowResponse struct {
	Message string `json:"message"`
}

// SyncActivityNow godoc
// @Summary      Trigger Activity Sync
// @Description  Manually triggers an immediate activity sync from the configured provider. Returns immediately; the sync runs in the background.
// @Tags         Activity
// @Produce      json
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse "Unauthorized (only when Auth.Enabled=true)"
// @Success      200  {object}  httpx.JSONResponse{data=syncNowResponse}
// @Failure      500  {object}  httpx.JSONResponse "Internal Server Error"
// @Router       /api/activity/sync [post]
func SyncActivityNow(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Manual Activity Sync", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)
	_ = ctx

	if !config.Current.ActivitySource.Enabled {
		logAction.SetError("Activity source not enabled", "Enable the activity source in settings before syncing", nil)
		httpx.SendResponse(w, ld, syncNowResponse{})
		return
	}

	go jobs.RunActivitySync()

	httpx.SendResponse(w, ld, syncNowResponse{
		Message: "Activity sync started",
	})
}
