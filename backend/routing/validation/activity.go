package routes_validation

import (
	"aura/activity"
	"aura/config"
	"aura/logging"
	"aura/utils/httpx"
	"net/http"
)

type ValidateActivitySource_Request struct {
	ActivitySource config.Config_ActivitySource `json:"activity_source"`
}

type ValidateActivitySource_Response struct {
	Valid    bool                    `json:"valid"`
	Message  string                  `json:"message"`
	Provider *activity.ProviderInfo  `json:"provider,omitempty"`
}

// ValidateActivitySourceInfo godoc
// @Summary      Validate Activity Source
// @Description  Test the connection to the configured activity provider (Tautulli or Tracearr).
// @Tags         Validation
// @Accept       json
// @Produce      json
// @Param        activity_source body ValidateActivitySource_Request true "Activity Source Configuration"
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse "Unauthorized (only when Auth.Enabled=true)"
// @Success      200  {object}  httpx.JSONResponse{data=ValidateActivitySource_Response}
// @Failure      500  {object}  httpx.JSONResponse "Internal Server Error"
// @Router       /api/validate/activity [post]
func ValidateActivitySourceInfo(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Validate Activity Source", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	var req ValidateActivitySource_Request
	var response ValidateActivitySource_Response

	Err := httpx.DecodeRequestBodyToJSON(ctx, r.Body, &req, "Activity Source Info")
	if Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	src := req.ActivitySource

	// If the token is masked, retrieve the stored token
	if config.IsMaskedField(src.ApiToken) {
		src.ApiToken = config.Current.ActivitySource.ApiToken
	}

	provider, err := activity.NewProvider(src.Provider)
	if err != nil {
		logAction.SetError("Unknown activity provider", err.Error(), map[string]any{"provider": src.Provider})
		httpx.SendResponse(w, ld, response)
		return
	}

	sourceCfg := activity.ActivitySourceConfig{
		Provider:         src.Provider,
		BaseURL:          src.BaseURL,
		ApiToken:         src.ApiToken,
		TracearrServerID: src.TracearrServerID,
	}

	info, err := provider.TestConnection(ctx, sourceCfg)
	if err != nil {
		logAction.SetError("Activity provider connection test failed", err.Error(), map[string]any{
			"provider": src.Provider,
			"url":      src.BaseURL,
		})
		httpx.SendResponse(w, ld, response)
		return
	}

	response.Valid = true
	response.Message = "Connection to " + info.Name + " successful"
	response.Provider = &info
	httpx.SendResponse(w, ld, response)
}
