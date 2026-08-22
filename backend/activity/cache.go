package activity

import (
	"sync"
	"time"
)

// flattenSummaries converts the aggregation map to a slice.
func flattenSummaries(agg map[string]*MediaActivitySummary) []MediaActivitySummary {
	out := make([]MediaActivitySummary, 0, len(agg))
	for _, s := range agg {
		out = append(out, *s)
	}
	return out
}

// SyncStatus tracks the outcome of the most recent sync run.
type SyncStatus struct {
	LastSuccessAt time.Time `json:"last_success_at,omitempty"`
	LastErrorAt   time.Time `json:"last_error_at,omitempty"`
	LastError     string    `json:"last_error,omitempty"`
	IsStale       bool      `json:"is_stale"`
}

// Cache holds the latest successfully synced summaries plus the sync status.
// A failed sync preserves the previous snapshot and marks it stale.
type Cache struct {
	mu        sync.RWMutex
	summaries []MediaActivitySummary
	status    SyncStatus
}

var DefaultCache = &Cache{}

// Update stores a fresh set of summaries from a successful sync.
func (c *Cache) Update(summaries []MediaActivitySummary) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.summaries = summaries
	c.status.LastSuccessAt = time.Now().UTC()
	c.status.LastError = ""
	c.status.LastErrorAt = time.Time{}
	c.status.IsStale = false
}

// SetError records a sync failure while preserving the previous snapshot.
func (c *Cache) SetError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.LastErrorAt = time.Now().UTC()
	c.status.LastError = err.Error()
	c.status.IsStale = true
}

// Summaries returns the latest (possibly stale) summaries.
func (c *Cache) Summaries() []MediaActivitySummary {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]MediaActivitySummary, len(c.summaries))
	copy(out, c.summaries)
	return out
}

// Status returns a copy of the current sync status.
func (c *Cache) Status() SyncStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}
