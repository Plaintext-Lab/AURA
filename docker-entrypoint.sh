#!/bin/sh
# Runs the backend and frontend, and passes docker stop to both so they can shut down cleanly

./main &
BACKEND_PID=$!

HOSTNAME=0.0.0.0 node server.js &
FRONTEND_PID=$!

stop() {
	kill -TERM "$BACKEND_PID" "$FRONTEND_PID" 2>/dev/null
}
trap stop TERM INT

# Runs until the frontend exits or a stop signal arrives
wait "$FRONTEND_PID"

# Stop both and wait for them to finish
stop
wait "$FRONTEND_PID" 2>/dev/null
wait "$BACKEND_PID" 2>/dev/null
