package models

import "time"

// LibraryGroup represents a user-defined group of compatible libraries that share artwork selections.
// All libraries in a group must have the same media type (movie or show).
type LibraryGroup struct {
	ID          string    `json:"id"`           // UUID stable identifier for the group
	Name        string    `json:"name"`         // Display name for the group
	MediaType   string    `json:"media_type"`   // "movie" or "show" – all member libraries must match
	LibraryIDs  []string  `json:"library_ids"`  // Stable library IDs of member libraries (min 2)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LibraryGroupPolicy is the group-level artwork policy for a specific media item.
// It is shared across every library in the group; individual per-library copies remain
// separate application targets.
type LibraryGroupPolicy struct {
	ID                string        `json:"id"`                  // UUID
	GroupID           string        `json:"group_id"`            // FK → LibraryGroup.ID
	TMDB_ID           string        `json:"tmdb_id"`             // TMDB ID of the media item
	Edition           string        `json:"edition"`             // Edition (movies only; "" for shows)
	SetID             string        `json:"set_id"`              // MediUX set ID
	SelectedTypes     SelectedTypes `json:"selected_types"`      // Which artwork types are active
	AutoDownload      bool          `json:"auto_download"`       // Auto-download flag
	FutureUpdatesOnly bool          `json:"future_updates_only"` // Apply only on future set updates
	LastReconciled    *time.Time    `json:"last_reconciled,omitempty"`
	ReconcileStatus   string        `json:"reconcile_status"` // "ok" | "partial" | "pending" | ""
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// LibraryGroupPreviewItem represents a single media item during group-creation reconciliation preview.
type LibraryGroupPreviewItem struct {
	TMDB_ID    string              `json:"tmdb_id"`
	Edition    string              `json:"edition"`
	Title      string              `json:"title"`
	Year       int                 `json:"year"`
	MediaType  string              `json:"media_type"`
	LibrarySets []LibraryItemSets  `json:"library_sets"` // one entry per library that has this item
	HasConflict bool               `json:"has_conflict"` // true when different libraries have different saved sets
}

// LibraryItemSets is the saved-set information for one library copy of a media item.
type LibraryItemSets struct {
	LibraryID    string       `json:"library_id"`
	LibraryTitle string       `json:"library_title"`
	SavedSets    []DBSavedSet `json:"saved_sets"`
}

// LibraryGroupPreview is returned by the preview endpoint before creating or editing a group.
type LibraryGroupPreview struct {
	GroupID       string                    `json:"group_id,omitempty"` // empty when previewing a new group
	MatchCount    int                       `json:"match_count"`
	ConflictCount int                       `json:"conflict_count"`
	Items         []LibraryGroupPreviewItem `json:"items"`
}
