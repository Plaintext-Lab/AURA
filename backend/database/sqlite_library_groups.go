package database

import (
	"aura/logging"
	"aura/models"
	"context"
	"database/sql"
)

func v7_CreateLibraryGroupsTable(ctx context.Context, conn *sql.DB) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Creating LibraryGroups Table", logging.LevelTrace)
	defer logAction.Complete()
	Err = logging.LogErrorInfo{}

	query := `
CREATE TABLE IF NOT EXISTS LibraryGroups (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	media_type TEXT NOT NULL CHECK (media_type IN ('movie','show')),
	library_ids TEXT NOT NULL DEFAULT '[]',
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
`
	_, err := conn.ExecContext(ctx, query)
	if err != nil {
		logAction.SetError("Failed to create LibraryGroups table", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	return Err
}

func v7_CreateLibraryGroupPoliciesTable(ctx context.Context, conn *sql.DB) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Creating LibraryGroupPolicies Table", logging.LevelTrace)
	defer logAction.Complete()
	Err = logging.LogErrorInfo{}

	query := `
CREATE TABLE IF NOT EXISTS LibraryGroupPolicies (
	id TEXT PRIMARY KEY,
	group_id TEXT NOT NULL,
	tmdb_id TEXT NOT NULL,
	edition TEXT NOT NULL DEFAULT '',
	set_id TEXT NOT NULL,

	poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (poster_selected IN (0,1)),
	backdrop_selected INTEGER NOT NULL DEFAULT 0 CHECK (backdrop_selected IN (0,1)),
	season_poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (season_poster_selected IN (0,1)),
	special_season_poster_selected INTEGER NOT NULL DEFAULT 0 CHECK (special_season_poster_selected IN (0,1)),
	titlecard_selected INTEGER NOT NULL DEFAULT 0 CHECK (titlecard_selected IN (0,1)),

	auto_download INTEGER NOT NULL DEFAULT 0 CHECK (auto_download IN (0,1)),
	future_updates_only INTEGER NOT NULL DEFAULT 0 CHECK (future_updates_only IN (0,1)),
	last_reconciled DATETIME,
	reconcile_status TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now')),

	UNIQUE (group_id, tmdb_id, edition),

	FOREIGN KEY (group_id) REFERENCES LibraryGroups(id)
		ON DELETE CASCADE
		ON UPDATE CASCADE
);
`
	_, err := conn.ExecContext(ctx, query)
	if err != nil {
		logAction.SetError("Failed to create LibraryGroupPolicies table", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	return Err
}

func v7_AddLibraryGroupIndexes(ctx context.Context, conn *sql.DB) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Adding LibraryGroup Indexes", logging.LevelTrace)
	defer logAction.Complete()
	Err = logging.LogErrorInfo{}

	queries := []string{
		`CREATE INDEX IF NOT EXISTS idx_librarygrouppolicies_group_id ON LibraryGroupPolicies(group_id);`,
		`CREATE INDEX IF NOT EXISTS idx_librarygrouppolicies_tmdb_id ON LibraryGroupPolicies(tmdb_id, edition);`,
	}
	for _, q := range queries {
		if _, err := conn.ExecContext(ctx, q); err != nil {
			logAction.SetError("Failed to add LibraryGroup index", err.Error(), map[string]any{"error": err.Error(), "query": q})
			return *logAction.Error
		}
	}
	return Err
}

// GetLibraryGroups returns all library groups.
func (s *SQliteDB) GetLibraryGroups(ctx context.Context) (groups []models.LibraryGroup, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Get Library Groups", logging.LevelDebug)
	defer logAction.Complete()

	rows, err := s.conn.QueryContext(ctx, `
		SELECT id, name, media_type, library_ids, created_at, updated_at
		FROM LibraryGroups ORDER BY name;
	`)
	if err != nil {
		logAction.SetError("Failed to query LibraryGroups", err.Error(), map[string]any{"error": err.Error()})
		return nil, *logAction.Error
	}
	defer rows.Close()

	for rows.Next() {
		var g models.LibraryGroup
		var libIDsJSON string
		if err := rows.Scan(&g.ID, &g.Name, &g.MediaType, &libIDsJSON, &g.CreatedAt, &g.UpdatedAt); err != nil {
			logAction.SetError("Failed to scan LibraryGroup row", err.Error(), map[string]any{"error": err.Error()})
			return nil, *logAction.Error
		}
		g.LibraryIDs = parseJSONStringSlice(libIDsJSON)
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []models.LibraryGroup{}
	}
	return groups, Err
}

// GetLibraryGroupByID returns a single library group by its ID.
func (s *SQliteDB) GetLibraryGroupByID(ctx context.Context, id string) (group models.LibraryGroup, found bool, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Get Library Group By ID", logging.LevelDebug)
	defer logAction.Complete()

	var libIDsJSON string
	err := s.conn.QueryRowContext(ctx, `
		SELECT id, name, media_type, library_ids, created_at, updated_at
		FROM LibraryGroups WHERE id = ?;
	`, id).Scan(&group.ID, &group.Name, &group.MediaType, &libIDsJSON, &group.CreatedAt, &group.UpdatedAt)
	if err == sql.ErrNoRows {
		return group, false, Err
	}
	if err != nil {
		logAction.SetError("Failed to query LibraryGroup by ID", err.Error(), map[string]any{"error": err.Error()})
		return group, false, *logAction.Error
	}
	group.LibraryIDs = parseJSONStringSlice(libIDsJSON)
	return group, true, Err
}

// UpsertLibraryGroup creates or updates a library group.
func (s *SQliteDB) UpsertLibraryGroup(ctx context.Context, g models.LibraryGroup) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Upsert Library Group", logging.LevelDebug)
	defer logAction.Complete()

	libIDsJSON := marshalJSONStringSlice(g.LibraryIDs)

	_, err := s.conn.ExecContext(ctx, `
		INSERT INTO LibraryGroups (id, name, media_type, library_ids, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			media_type = excluded.media_type,
			library_ids = excluded.library_ids,
			updated_at = datetime('now');
	`, g.ID, g.Name, g.MediaType, libIDsJSON)
	if err != nil {
		logAction.SetError("Failed to upsert LibraryGroup", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	return Err
}

// DeleteLibraryGroup removes a library group and all its policies by ID.
func (s *SQliteDB) DeleteLibraryGroup(ctx context.Context, id string) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Delete Library Group", logging.LevelDebug)
	defer logAction.Complete()

	_, err := s.conn.ExecContext(ctx, `DELETE FROM LibraryGroups WHERE id = ?;`, id)
	if err != nil {
		logAction.SetError("Failed to delete LibraryGroup", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	return Err
}

// GetLibraryGroupPolicies returns all policies for the given group.
func (s *SQliteDB) GetLibraryGroupPolicies(ctx context.Context, groupID string) (policies []models.LibraryGroupPolicy, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Get Library Group Policies", logging.LevelDebug)
	defer logAction.Complete()

	rows, err := s.conn.QueryContext(ctx, `
		SELECT id, group_id, tmdb_id, edition, set_id,
		       poster_selected, backdrop_selected, season_poster_selected, special_season_poster_selected, titlecard_selected,
		       auto_download, future_updates_only, last_reconciled, reconcile_status, created_at, updated_at
		FROM LibraryGroupPolicies WHERE group_id = ?;
	`, groupID)
	if err != nil {
		logAction.SetError("Failed to query LibraryGroupPolicies", err.Error(), map[string]any{"error": err.Error()})
		return nil, *logAction.Error
	}
	defer rows.Close()

	for rows.Next() {
		var p models.LibraryGroupPolicy
		var lastReconciled *string
		err := rows.Scan(
			&p.ID, &p.GroupID, &p.TMDB_ID, &p.Edition, &p.SetID,
			&p.SelectedTypes.Poster, &p.SelectedTypes.Backdrop,
			&p.SelectedTypes.SeasonPoster, &p.SelectedTypes.SpecialSeasonPoster, &p.SelectedTypes.Titlecard,
			&p.AutoDownload, &p.FutureUpdatesOnly, &lastReconciled, &p.ReconcileStatus,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			logAction.SetError("Failed to scan LibraryGroupPolicy row", err.Error(), map[string]any{"error": err.Error()})
			return nil, *logAction.Error
		}
		if lastReconciled != nil {
			t, _ := parseDateTime(*lastReconciled)
			p.LastReconciled = &t
		}
		policies = append(policies, p)
	}
	if policies == nil {
		policies = []models.LibraryGroupPolicy{}
	}
	return policies, Err
}

// GetLibraryGroupPolicyByTMDB returns the policy for a specific TMDB item within a group.
func (s *SQliteDB) GetLibraryGroupPolicyByTMDB(ctx context.Context, groupID, tmdbID, edition string) (policy models.LibraryGroupPolicy, found bool, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Get Library Group Policy By TMDB", logging.LevelDebug)
	defer logAction.Complete()

	var lastReconciled *string
	err := s.conn.QueryRowContext(ctx, `
		SELECT id, group_id, tmdb_id, edition, set_id,
		       poster_selected, backdrop_selected, season_poster_selected, special_season_poster_selected, titlecard_selected,
		       auto_download, future_updates_only, last_reconciled, reconcile_status, created_at, updated_at
		FROM LibraryGroupPolicies WHERE group_id = ? AND tmdb_id = ? AND edition = ?;
	`, groupID, tmdbID, edition).Scan(
		&policy.ID, &policy.GroupID, &policy.TMDB_ID, &policy.Edition, &policy.SetID,
		&policy.SelectedTypes.Poster, &policy.SelectedTypes.Backdrop,
		&policy.SelectedTypes.SeasonPoster, &policy.SelectedTypes.SpecialSeasonPoster, &policy.SelectedTypes.Titlecard,
		&policy.AutoDownload, &policy.FutureUpdatesOnly, &lastReconciled, &policy.ReconcileStatus,
		&policy.CreatedAt, &policy.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return policy, false, Err
	}
	if err != nil {
		logAction.SetError("Failed to query LibraryGroupPolicy by TMDB", err.Error(), map[string]any{"error": err.Error()})
		return policy, false, *logAction.Error
	}
	if lastReconciled != nil {
		t, _ := parseDateTime(*lastReconciled)
		policy.LastReconciled = &t
	}
	return policy, true, Err
}

// UpsertLibraryGroupPolicy creates or updates a group-level artwork policy.
func (s *SQliteDB) UpsertLibraryGroupPolicy(ctx context.Context, p models.LibraryGroupPolicy) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Upsert Library Group Policy", logging.LevelDebug)
	defer logAction.Complete()

	_, err := s.conn.ExecContext(ctx, `
		INSERT INTO LibraryGroupPolicies (
			id, group_id, tmdb_id, edition, set_id,
			poster_selected, backdrop_selected, season_poster_selected, special_season_poster_selected, titlecard_selected,
			auto_download, future_updates_only, last_reconciled, reconcile_status,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(group_id, tmdb_id, edition) DO UPDATE SET
			set_id = excluded.set_id,
			poster_selected = excluded.poster_selected,
			backdrop_selected = excluded.backdrop_selected,
			season_poster_selected = excluded.season_poster_selected,
			special_season_poster_selected = excluded.special_season_poster_selected,
			titlecard_selected = excluded.titlecard_selected,
			auto_download = excluded.auto_download,
			future_updates_only = excluded.future_updates_only,
			last_reconciled = excluded.last_reconciled,
			reconcile_status = excluded.reconcile_status,
			updated_at = datetime('now');
	`,
		p.ID, p.GroupID, p.TMDB_ID, p.Edition, p.SetID,
		boolToInt(p.SelectedTypes.Poster), boolToInt(p.SelectedTypes.Backdrop),
		boolToInt(p.SelectedTypes.SeasonPoster), boolToInt(p.SelectedTypes.SpecialSeasonPoster),
		boolToInt(p.SelectedTypes.Titlecard),
		boolToInt(p.AutoDownload), boolToInt(p.FutureUpdatesOnly),
		nullableDateTime(p.LastReconciled), p.ReconcileStatus,
	)
	if err != nil {
		logAction.SetError("Failed to upsert LibraryGroupPolicy", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	return Err
}

// DeleteLibraryGroupPolicy removes a group policy by group + tmdb + edition.
func (s *SQliteDB) DeleteLibraryGroupPolicy(ctx context.Context, groupID, tmdbID, edition string) (Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Delete Library Group Policy", logging.LevelDebug)
	defer logAction.Complete()

	_, err := s.conn.ExecContext(ctx, `DELETE FROM LibraryGroupPolicies WHERE group_id = ? AND tmdb_id = ? AND edition = ?;`, groupID, tmdbID, edition)
	if err != nil {
		logAction.SetError("Failed to delete LibraryGroupPolicy", err.Error(), map[string]any{"error": err.Error()})
		return *logAction.Error
	}
	return Err
}

// GetGroupsForLibrary returns all groups that contain the given library ID.
func (s *SQliteDB) GetGroupsForLibrary(ctx context.Context, libraryID string) (groups []models.LibraryGroup, Err logging.LogErrorInfo) {
	ctx, logAction := logging.AddSubActionToContext(ctx, "Get Groups For Library", logging.LevelDebug)
	defer logAction.Complete()

	rows, err := s.conn.QueryContext(ctx, `
		SELECT id, name, media_type, library_ids, created_at, updated_at
		FROM LibraryGroups
		WHERE EXISTS (SELECT 1 FROM json_each(library_ids) WHERE value = ?);
	`, libraryID)
	if err != nil {
		logAction.SetError("Failed to query groups for library", err.Error(), map[string]any{"error": err.Error()})
		return nil, *logAction.Error
	}
	defer rows.Close()

	for rows.Next() {
		var g models.LibraryGroup
		var libIDsJSON string
		if err := rows.Scan(&g.ID, &g.Name, &g.MediaType, &libIDsJSON, &g.CreatedAt, &g.UpdatedAt); err != nil {
			logAction.SetError("Failed to scan LibraryGroup row", err.Error(), map[string]any{"error": err.Error()})
			return nil, *logAction.Error
		}
		g.LibraryIDs = parseJSONStringSlice(libIDsJSON)
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []models.LibraryGroup{}
	}
	return groups, Err
}
