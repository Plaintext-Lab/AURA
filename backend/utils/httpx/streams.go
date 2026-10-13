package httpx

import "sync"

var (
	streamsClosed    = make(chan struct{})
	closeStreamsOnce sync.Once
)

// CloseStreams ends the open event streams, so shutting down doesn't wait for them
func CloseStreams() {
	closeStreamsOnce.Do(func() { close(streamsClosed) })
}

// StreamsClosed is closed when aura shuts down
func StreamsClosed() <-chan struct{} {
	return streamsClosed
}
