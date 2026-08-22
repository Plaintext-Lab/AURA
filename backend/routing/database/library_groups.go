package routes_db

import (
	"aura/cache"
	"aura/config"
	"aura/database"
	"aura/logging"
	"aura/models"
	"aura/utils/httpx"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// ─────────────────────────────────────────────────────────────────────────────
// Request / response shapes
// ─────────────────────────────────────────────────────────────────────────────

type libraryGroupsResponse struct {
	Groups []models.LibraryGroup `json:"groups"`
}

type libraryGroupResponse struct {
	Group models.LibraryGroup `json:"group"`
}

type upsertLibraryGroupRequest struct {
	Name       string   `json:"name"`
	MediaType  string   `json:"media_type"`
	LibraryIDs []string `json:"library_ids"`
}

type libraryGroupPreviewResponse struct {
	Preview models.LibraryGroupPreview `json:"preview"`
}

type reconcileLibraryGroupRequest struct {
	// PolicyDecisions maps "<tmdb_id>|<edition>" → set_id to use for the group policy.
	// Items not present in the map are left unlinked (no group policy).
	PolicyDecisions map[string]reconcileDecision `json:"policy_decisions"`
}

type reconcileDecision struct {
	SetID         string              `json:"set_id"`
	SelectedTypes models.SelectedTypes `json:"selected_types"`
	AutoDownload  bool                `json:"auto_download"`
}

type reconcileLibraryGroupResponse struct {
	Reconciled int `json:"reconciled"`
	Skipped    int `json:"skipped"`
}

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/db/library-groups
// ─────────────────────────────────────────────────────────────────────────────

// GetLibraryGroups godoc
// @Summary      List Library Groups
// @Description  Return all linked-library groups.
// @Tags         Database
// @Produce      json
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse
// @Success      200  {object}  httpx.JSONResponse{data=libraryGroupsResponse}
// @Failure      500  {object}  httpx.JSONResponse
// @Router       /api/db/library-groups [get]
func GetLibraryGroups(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Get Library Groups", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	var response libraryGroupsResponse

	groups, Err := database.GetLibraryGroups(ctx)
	if Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}
	response.Groups = groups
	httpx.SendResponse(w, ld, response)
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/db/library-groups
// ─────────────────────────────────────────────────────────────────────────────

// CreateLibraryGroup godoc
// @Summary      Create Library Group
// @Description  Create a new linked-library group.
// @Tags         Database
// @Accept       json
// @Produce      json
// @Param        req  body      upsertLibraryGroupRequest  true  "Create Group Request"
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse
// @Success      200  {object}  httpx.JSONResponse{data=libraryGroupResponse}
// @Failure      500  {object}  httpx.JSONResponse
// @Router       /api/db/library-groups [post]
func CreateLibraryGroup(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Create Library Group", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	var req upsertLibraryGroupRequest
	var response libraryGroupResponse

	if Err := httpx.DecodeRequestBodyToJSON(ctx, r.Body, &req, "Create Library Group - Decode Body"); Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	if validErr := validateLibraryGroupRequest(logAction, req); validErr.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	g := models.LibraryGroup{
		ID:         newRandomID(),
		Name:       req.Name,
		MediaType:  req.MediaType,
		LibraryIDs: req.LibraryIDs,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if Err := database.UpsertLibraryGroup(ctx, g); Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	response.Group = g
	httpx.SendResponse(w, ld, response)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUT /api/db/library-groups/{id}
// ─────────────────────────────────────────────────────────────────────────────

// UpdateLibraryGroup godoc
// @Summary      Update Library Group
// @Description  Update an existing linked-library group.
// @Tags         Database
// @Accept       json
// @Produce      json
// @Param        id   path      string                     true  "Group ID"
// @Param        req  body      upsertLibraryGroupRequest  true  "Update Group Request"
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse
// @Success      200  {object}  httpx.JSONResponse{data=libraryGroupResponse}
// @Failure      500  {object}  httpx.JSONResponse
// @Router       /api/db/library-groups/{id} [put]
func UpdateLibraryGroup(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Update Library Group", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	id := chi.URLParam(r, "id")
	var req upsertLibraryGroupRequest
	var response libraryGroupResponse

	if Err := httpx.DecodeRequestBodyToJSON(ctx, r.Body, &req, "Update Library Group - Decode Body"); Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	existing, found, Err := database.GetLibraryGroupByID(ctx, id)
	if Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}
	if !found {
		logAction.SetError("Library Group not found", fmt.Sprintf("No group with ID %s", id), map[string]any{"id": id})
		httpx.SendResponse(w, ld, response)
		return
	}

	if validErr := validateLibraryGroupRequest(logAction, req); validErr.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	existing.Name = req.Name
	existing.MediaType = req.MediaType
	existing.LibraryIDs = req.LibraryIDs
	existing.UpdatedAt = time.Now()

	if Err := database.UpsertLibraryGroup(ctx, existing); Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	response.Group = existing
	httpx.SendResponse(w, ld, response)
}

// ─────────────────────────────────────────────────────────────────────────────
// DELETE /api/db/library-groups/{id}
// ─────────────────────────────────────────────────────────────────────────────

// DeleteLibraryGroup godoc
// @Summary      Delete Library Group
// @Description  Delete a linked-library group. Existing applied artwork is preserved.
// @Tags         Database
// @Produce      json
// @Param        id   path      string  true  "Group ID"
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse
// @Success      200  {object}  httpx.JSONResponse{data=map[string]string}
// @Failure      500  {object}  httpx.JSONResponse
// @Router       /api/db/library-groups/{id} [delete]
func DeleteLibraryGroup(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Delete Library Group", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	id := chi.URLParam(r, "id")

	if Err := database.DeleteLibraryGroup(ctx, id); Err.Message != "" {
		httpx.SendResponse(w, ld, map[string]string{"result": "error"})
		return
	}
	httpx.SendResponse(w, ld, map[string]string{"result": "ok"})
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/db/library-groups/{id}/preview
// ─────────────────────────────────────────────────────────────────────────────

// PreviewLibraryGroup godoc
// @Summary      Preview Library Group Matches and Conflicts
// @Description  Show matching items and conflicting artwork selections for a group before creating or editing it.
// @Tags         Database
// @Accept       json
// @Produce      json
// @Param        id   path      string                     true  "Group ID (use 'new' when previewing before creation)"
// @Param        req  body      upsertLibraryGroupRequest  true  "Group definition to preview"
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse
// @Success      200  {object}  httpx.JSONResponse{data=libraryGroupPreviewResponse}
// @Failure      500  {object}  httpx.JSONResponse
// @Router       /api/db/library-groups/{id}/preview [post]
func PreviewLibraryGroup(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Preview Library Group", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	id := chi.URLParam(r, "id")
	var req upsertLibraryGroupRequest
	var response libraryGroupPreviewResponse

	if Err := httpx.DecodeRequestBodyToJSON(ctx, r.Body, &req, "Preview Library Group - Decode Body"); Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	preview, Err := buildLibraryGroupPreview(ctx, id, req)
	if Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	response.Preview = preview
	httpx.SendResponse(w, ld, response)
}

// ─────────────────────────────────────────────────────────────────────────────
// POST /api/db/library-groups/{id}/reconcile
// ─────────────────────────────────────────────────────────────────────────────

// ReconcileLibraryGroup godoc
// @Summary      Reconcile Library Group
// @Description  Apply policy decisions from a reconciliation preview.
// @Tags         Database
// @Accept       json
// @Produce      json
// @Param        id   path      string                        true  "Group ID"
// @Param        req  body      reconcileLibraryGroupRequest  true  "Reconcile Request"
// @Security     SessionCookie
// @Security     ApiKeyAuth
// @Failure      401  {object}  httpx.UnauthorizedResponse
// @Success      200  {object}  httpx.JSONResponse{data=reconcileLibraryGroupResponse}
// @Failure      500  {object}  httpx.JSONResponse
// @Router       /api/db/library-groups/{id}/reconcile [post]
func ReconcileLibraryGroup(w http.ResponseWriter, r *http.Request) {
	ctx, ld := logging.CreateLoggingContext(r.Context(), r.URL.Path)
	logAction := ld.AddAction("Reconcile Library Group", logging.LevelInfo)
	ctx = logging.WithCurrentAction(ctx, logAction)

	id := chi.URLParam(r, "id")
	var req reconcileLibraryGroupRequest
	var response reconcileLibraryGroupResponse

	if Err := httpx.DecodeRequestBodyToJSON(ctx, r.Body, &req, "Reconcile Library Group - Decode Body"); Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}

	_, found, Err := database.GetLibraryGroupByID(ctx, id)
	if Err.Message != "" {
		httpx.SendResponse(w, ld, response)
		return
	}
	if !found {
		logAction.SetError("Library Group not found", fmt.Sprintf("No group with ID %s", id), map[string]any{"id": id})
		httpx.SendResponse(w, ld, response)
		return
	}

	now := time.Now()
	for key, decision := range req.PolicyDecisions {
		tmdbID, edition := splitTMDBKey(key)
		if tmdbID == "" || decision.SetID == "" {
			response.Skipped++
			continue
		}

		existingPolicy, _, _ := database.GetLibraryGroupPolicyByTMDB(ctx, id, tmdbID, edition)
		policyID := existingPolicy.ID
		if policyID == "" {
			policyID = newRandomID()
		}

		p := models.LibraryGroupPolicy{
			ID:              policyID,
			GroupID:         id,
			TMDB_ID:         tmdbID,
			Edition:         edition,
			SetID:           decision.SetID,
			SelectedTypes:   decision.SelectedTypes,
			AutoDownload:    decision.AutoDownload,
			LastReconciled:  &now,
			ReconcileStatus: "ok",
		}
		if Err := database.UpsertLibraryGroupPolicy(ctx, p); Err.Message != "" {
			response.Skipped++
			continue
		}
		response.Reconciled++
	}

	httpx.SendResponse(w, ld, response)
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// validateLibraryGroupRequest checks the common constraints for create/update.
func validateLibraryGroupRequest(logAction *logging.LogAction, req upsertLibraryGroupRequest) logging.LogErrorInfo {
	if req.Name == "" {
		logAction.SetError("Invalid Library Group", "Name is required", nil)
		return *logAction.Error
	}
	if req.MediaType != "movie" && req.MediaType != "show" {
		logAction.SetError("Invalid Library Group", "media_type must be 'movie' or 'show'", nil)
		return *logAction.Error
	}
	if len(req.LibraryIDs) < 2 {
		logAction.SetError("Invalid Library Group", "A group must contain at least two libraries", nil)
		return *logAction.Error
	}
	// All IDs must refer to libraries of the same media type as the group
	libs := config.Current.MediaServer.Libraries
	for _, lid := range req.LibraryIDs {
		found := false
		for _, lib := range libs {
			if lib.ID == lid {
				found = true
				if lib.Type != req.MediaType {
					logAction.SetError("Invalid Library Group",
						fmt.Sprintf("Library %s (type %s) is not compatible with group media type %s", lid, lib.Type, req.MediaType),
						map[string]any{"library_id": lid})
					return *logAction.Error
				}
				break
			}
		}
		if !found {
			logAction.SetError("Invalid Library Group",
				fmt.Sprintf("Library ID %s not found in configuration", lid),
				map[string]any{"library_id": lid})
			return *logAction.Error
		}
	}
	return logging.LogErrorInfo{}
}

// buildLibraryGroupPreview constructs a reconciliation preview for a proposed group.
func buildLibraryGroupPreview(ctx context.Context, groupID string, req upsertLibraryGroupRequest) (models.LibraryGroupPreview, logging.LogErrorInfo) {
	preview := models.LibraryGroupPreview{GroupID: groupID}

	// Build a map of libraryID → libraryTitle from config
	libTitleByID := map[string]string{}
	for _, lib := range config.Current.MediaServer.Libraries {
		libTitleByID[lib.ID] = lib.Title
	}

	// Build a map: tmdbID+edition → LibraryGroupPreviewItem
	itemMap := map[string]*models.LibraryGroupPreviewItem{}

	for _, lid := range req.LibraryIDs {
		// Look up the title directly; skip this library if it's not in config
		libTitle := libTitleByID[lid]
		if libTitle == "" {
			continue
		}

		// Get all media items for this library from cache
		section, found := cache.LibraryStore.GetSectionByTitle(libTitle)
		if !found || section == nil {
			continue
		}

		for _, mi := range section.MediaItems {
			// Only match by type + TMDB ID; require a TMDB ID
			if mi.TMDB_ID == "" {
				continue
			}
			// For movies: match type + TMDB ID + edition
			// For shows: match type + TMDB ID
			key := fmt.Sprintf("%s|%s", mi.TMDB_ID, mi.Edition)
			if mi.Type == "show" {
				key = fmt.Sprintf("%s|", mi.TMDB_ID)
			}

			if _, exists := itemMap[key]; !exists {
				itemMap[key] = &models.LibraryGroupPreviewItem{
					TMDB_ID:     mi.TMDB_ID,
					Edition:     mi.Edition,
					Title:       mi.Title,
					Year:        mi.Year,
					MediaType:   mi.Type,
					LibrarySets: []models.LibraryItemSets{},
				}
			}
			itemMap[key].LibrarySets = append(itemMap[key].LibrarySets, models.LibraryItemSets{
				LibraryID:    lid,
				LibraryTitle: libTitle,
				SavedSets:    mi.DBSavedSets,
			})
		}
	}

	items := make([]models.LibraryGroupPreviewItem, 0, len(itemMap))
	conflictCount := 0
	matchCount := 0

	for _, item := range itemMap {
		// Only include items present in 2+ libraries
		if len(item.LibrarySets) < 2 {
			continue
		}
		matchCount++

		// Detect conflict: different set IDs across libraries
		firstSetID := ""
		hasConflict := false
		for _, ls := range item.LibrarySets {
			for _, s := range ls.SavedSets {
				if firstSetID == "" {
					firstSetID = s.ID
				} else if s.ID != firstSetID {
					hasConflict = true
					break
				}
			}
			if hasConflict {
				break
			}
		}
		item.HasConflict = hasConflict
		if hasConflict {
			conflictCount++
		}
		items = append(items, *item)
	}

	preview.MatchCount = matchCount
	preview.ConflictCount = conflictCount
	preview.Items = items
	return preview, logging.LogErrorInfo{}
}

// splitTMDBKey reverses the "<tmdb_id>|<edition>" encoding used as policy map keys.
func splitTMDBKey(key string) (tmdbID, edition string) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '|' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

// newRandomID generates a random hex string suitable for use as a database ID.
func newRandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback: timestamp-based
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
