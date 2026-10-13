package routes_config

import (
	"aura/config"
	"aura/logging"
	"aura/utils/httpx"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const statusKeepAliveInterval = 25 * time.Second

// StreamAppConfigStatus godoc
// @Summary      Stream Config Status
// @Description  Server-Sent Events stream of the app status. Sends the status on connect, then again whenever it changes, like each step while aura starts.
// @Tags         Config
// @Produce      text/event-stream
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Router       /api/config/stream [get]
func StreamAppConfigStatus(w http.ResponseWriter, r *http.Request) {
	streamConfigStatus(w, r, currentConfigStatus)
}

// StreamSetupConfigStatus godoc
// @Summary      Stream Config Status (Setup Mode)
// @Description  Server-Sent Events stream of the app status while aura starts or is in setup mode. Without the setup code, only the app status is sent, not the config.
// @Tags         Config
// @Produce      text/event-stream
// @Router       /api/config/stream [get]
func StreamSetupConfigStatus(w http.ResponseWriter, r *http.Request) {
	streamConfigStatus(w, r, setupConfigStatus)
}

func streamConfigStatus(w http.ResponseWriter, r *http.Request, status func(ctx context.Context) AppConfigStatus) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Stream Config Status", logging.LevelTrace)
	ctx = logging.WithCurrentAction(ctx, logAction)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	changes := config.SubscribeStatus()
	defer config.UnsubscribeStatus(changes)

	send := func() {
		data, err := json.Marshal(configStatusResponse{Status: status(ctx)})
		if err != nil {
			logging.LOGGER.Error().Timestamp().Err(err).Msg("Failed to marshal the config status event")
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
	send()

	keepAlive := time.NewTicker(statusKeepAliveInterval)
	defer keepAlive.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-httpx.StreamsClosed():
			return
		case <-changes:
			send()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}
