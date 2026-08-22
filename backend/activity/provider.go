package activity

import (
	"context"
	"fmt"
	"time"
)

// MediaType represents the type of a media item tracked by an activity provider.
type MediaType string

const (
	MediaTypeMovie  MediaType = "movie"
	MediaTypeShow   MediaType = "show"
	MediaTypeEpisode MediaType = "episode"
)

// ActivitySourceConfig holds the provider-neutral configuration required to contact
// an activity provider. Secrets are kept here and must never be logged or returned
// to API callers without masking.
type ActivitySourceConfig struct {
	Provider        string `json:"provider" yaml:"Provider"`                               // "tautulli" or "tracearr"
	BaseURL         string `json:"base_url" yaml:"BaseURL"`                                // Base URL of the provider
	ApiToken        string `json:"api_token" yaml:"ApiToken"`                              // API key / ****** (treated as a secret)
	TracearrServerID string `json:"tracearr_server_id,omitempty" yaml:"TracearrServerID,omitempty"` // Tracearr: which server UUID to query
}

// ProviderInfo is returned by TestConnection and describes the remote provider.
type ProviderInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	ServerID string `json:"server_id,omitempty"` // Tracearr server UUID
}

// ActivitySyncRequest describes the parameters for a single sync run.
type ActivitySyncRequest struct {
	Source ActivitySourceConfig
	// WindowDays is how many days of history to fetch (e.g. 30).
	WindowDays int
	// MaxPages bounds pagination to prevent runaway requests.
	MaxPages int
}

// MediaActivitySummary is the normalised, privacy-safe aggregate record that
// AURA stores and exposes. All user/device/IP fields are discarded by adapters
// before producing this struct.
type MediaActivitySummary struct {
	Source          string    `json:"source"`                      // Provider name
	LibraryID       string    `json:"library_id"`                  // AURA stable library ID
	RatingKey       string    `json:"rating_key"`                  // Media-server rating key
	GrandparentKey  string    `json:"grandparent_key,omitempty"`   // Parent show rating key for episodes
	MediaType       MediaType `json:"media_type"`                  // movie | show | episode
	TmdbID          string    `json:"tmdb_id,omitempty"`           // TMDB ID if supplied
	TvdbID          string    `json:"tvdb_id,omitempty"`           // TVDB ID if supplied
	Title           string    `json:"title"`                       // Display title (not used for matching)
	PlayCount       int       `json:"play_count"`                  // Number of plays in the activity window
	WatchTimeSeconds int64    `json:"watch_time_seconds"`          // Aggregate play duration in seconds
	LastWatched     time.Time `json:"last_watched"`                // Timestamp of most recent play
	WindowStart     time.Time `json:"window_start"`                // Start of the aggregation window
	WindowEnd       time.Time `json:"window_end"`                  // End of the aggregation window
}

// ActivityProvider is the single interface that all activity adapters must implement.
// Adapters must discard any user, device, player, IP and location fields from
// provider responses before returning results.
type ActivityProvider interface {
	// TestConnection verifies that the provider is reachable and the credentials
	// are valid. It returns basic information about the remote service.
	TestConnection(ctx context.Context, cfg ActivitySourceConfig) (ProviderInfo, error)

	// Sync fetches activity for the window described in the request and returns
	// aggregate, privacy-safe summaries. Adapters must paginate internally and
	// must not exceed request.MaxPages pages.
	Sync(ctx context.Context, request ActivitySyncRequest) ([]MediaActivitySummary, error)
}

// NewProvider returns the ActivityProvider implementation for the named provider.
// Returns an error for unknown provider names.
func NewProvider(providerName string) (ActivityProvider, error) {
	switch providerName {
	case "tautulli":
		return &TautulliAdapter{}, nil
	case "tracearr":
		return &TracearrAdapter{}, nil
	default:
		return nil, fmt.Errorf("unknown activity provider %q: supported providers are tautulli, tracearr", providerName)
	}
}
