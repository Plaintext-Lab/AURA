package routes_download

import (
	"aura/database"
	downloadqueue "aura/download/queue"
	"aura/logging"
	"aura/utils/httpx"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const sseKeepAliveInterval = 25 * time.Second

// StreamDownloadQueueEvents godoc
// @Summary      Download Queue - Stream Events
// @Description  Server-Sent Events stream of live download queue updates (job added/started/finished/removed). Sends a full snapshot immediately on connect.
// @Tags         Download
// @Produce      text/event-stream
// @Security     SessionCookie
// @Router       /api/download/queue/stream [get]
func StreamDownloadQueueEvents(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Download Queue - Stream Events", logging.LevelInfo)
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

	ch := downloadqueue.QueueBroadcaster.Subscribe()
	defer downloadqueue.QueueBroadcaster.Unsubscribe(ch)

	jobs, Err := database.GetAllDownloadQueueJobs(ctx)
	if Err.Message != "" {
		jobs = []database.DownloadQueueJob{}
	}
	writeSSEEvent(w, flusher, downloadqueue.QueueEvent{Type: "snapshot", Jobs: jobs})

	keepAlive := time.NewTicker(sseKeepAliveInterval)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-httpx.StreamsClosed():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, evt)
		case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, evt downloadqueue.QueueEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		logging.LOGGER.Error().Timestamp().Err(err).Msg("Failed to marshal Download Queue SSE event")
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
