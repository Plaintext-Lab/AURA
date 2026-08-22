package routes_activity

import (
	"aura/activity"
	"aura/database"
	"aura/logging"
	"aura/utils/httpx"
	"net/http"
)

type activitySummariesResponse struct {
	Summaries []database.ActivitySummaryRow `json:"summaries"`
	Status    activity.SyncStatus           `json:"status"`
}

// GetActivitySummaries godoc
// @Summary      Get Activity Summaries
// @Description  Returns aggregate media activity summaries from the configured activity provider. Summaries contain no user, device or IP information.
// @Tags         Activity
// @Produce      json
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse "Unauthorized (only when Auth.Enabled=true)"
// @Success      200  {object}  httpx.JSONResponse{data=activitySummariesResponse}
// @Failure      500  {object}  httpx.JSONResponse "Internal Server Error"
// @Router       /api/activity/summaries [get]
func GetActivitySummaries(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Get Activity Summaries", logging.LevelTrace)
	ctx = logging.WithCurrentAction(ctx, logAction)

	rows, dbErr := database.GetActivitySummaries(ctx)
	if dbErr.Message != "" {
		httpx.SendResponse(w, ld, activitySummariesResponse{})
		return
	}

	if rows == nil {
		rows = []database.ActivitySummaryRow{}
	}

	httpx.SendResponse(w, ld, activitySummariesResponse{
		Summaries: rows,
		Status:    activity.DefaultCache.Status(),
	})
}
