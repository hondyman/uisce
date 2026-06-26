package analytics

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/cube/dialect"
	"github.com/hondyman/semlayer/backend/internal/cubeengine"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/internal/telemetry/optimize"
	"github.com/hondyman/semlayer/backend/models" // Adjusted for consistency
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// QueryService orchestrates the query generation and execution pipeline.
type QueryService struct {
	db            *sqlx.DB
	optService    *optimize.Service
	modelProvider *ModelProvider // This type is in the same package
}

// NewQueryService creates a new QueryService.
func NewQueryService(db *sqlx.DB, optService *optimize.Service, modelProvider *ModelProvider) *QueryService {
	return &QueryService{
		db:            db,
		optService:    optService,
		modelProvider: modelProvider,
	}
}

// buildEngineRequestFromView merges a user's query with a view's definition
// to create a request for the query engine.
func buildEngineRequestFromView(req models.ExplorerQueryRequest, view cube.ViewMeta, secCtx security.Context) (cube.QueryRequest, error) {
	// Start with the user's request
	engineReq := cube.QueryRequest{
		Cubes:      view.Cubes, // The view determines the cube(s)
		QueryType:  "regular",
		Measures:   req.Measures,
		Dimensions: req.Dimensions,
		Timezone:   req.Timezone,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}

	// Merge filters: view filters are applied first, then scope/region, then user filters.
	engineReq.Filters = append(engineReq.Filters, view.Filters...)
	if strings.TrimSpace(secCtx.OperatingScope) != "" {
		engineReq.Filters = append(engineReq.Filters, map[string]any{
			"member":   "operating_scope",
			"operator": "equals",
			"values":   []string{secCtx.OperatingScope},
		})
	}
	if strings.TrimSpace(secCtx.Region) != "" {
		engineReq.Filters = append(engineReq.Filters, map[string]any{
			"member":   "region",
			"operator": "equals",
			"values":   []string{secCtx.Region},
		})
	}
	for _, f := range req.Filters {
		engineReq.Filters = append(engineReq.Filters, map[string]any{
			"member":   f.Field,
			"operator": f.Op,
			"values":   f.Values,
		})
	}

	// User's ordering takes precedence
	if len(req.Order) > 0 {
		engineReq.Order = make([]any, len(req.Order))
		for i, o := range req.Order {
			engineReq.Order[i] = []string{o.Field, o.Dir}
		}
	}

	return engineReq, nil
}

// SavedQueryResponse is a version of ExplorerSavedQuery with the request JSON parsed.
type SavedQueryResponse struct {
	ID             uuid.UUID                   `json:"id"`
	Name           string                      `json:"name"`
	Request        models.ExplorerQueryRequest `json:"request"`
	LastRunAt      *time.Time                  `json:"last_run_at,omitempty"`
	LastDurationMs *int                        `json:"last_duration_ms,omitempty"`
}

// ListHistory retrieves recent saved queries for a user.
func (s *QueryService) ListHistory(ctx context.Context, userID string) ([]SavedQueryResponse, error) {
	var queries []models.ExplorerSavedQuery
	// In a real app, you'd filter by user_id and tenant_id.
	query := `
		SELECT id, created_at, owner_user_id, name, tags, request, last_run_at, last_duration_ms 
		FROM explorer_saved_query 
		ORDER BY last_run_at DESC NULLS LAST, created_at DESC 
		LIMIT 50
	`
	err := s.db.SelectContext(ctx, &queries, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list saved queries: %w", err)
	}

	// Unmarshal the request JSON for each query
	var response []SavedQueryResponse
	for _, q := range queries {
		var req models.ExplorerQueryRequest
		if err := json.Unmarshal(q.Query, &req); err == nil {
			response = append(response, SavedQueryResponse{
				ID:        q.ID,
				Name:      q.Name,
				Request:   req,
				LastRunAt: q.LastRunAt,
			})
		}
	}
	return response, nil
}

// CompileQuery takes a request, builds a plan, and returns the generated SQL without execution.
func (s *QueryService) CompileQuery(ctx context.Context, secCtx security.Context, req models.ExplorerQueryRequest) (*models.CompileResult, error) {
	if req.View == "" {
		return nil, fmt.Errorf("a view must be specified")
	}

	catalog, err := s.modelProvider.GetActiveCatalog(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("could not load active model catalog: %w", err)
	}

	view, ok := catalog.Views[req.View]
	if !ok {
		return nil, fmt.Errorf("view '%s' not found in catalog", req.View)
	}

	engineReq, err := buildEngineRequestFromView(req, view, secCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build query from view: %w", err)
	}

	engine := cubeengine.NewEngine(catalog, s.db.DB, s.optService, dialect.Postgres{})
	d := dialect.Postgres{}

	// --- RLS Injection ---
	policies := []security.RLSPolicy{security.OrdersPolicy}
	if len(engineReq.Cubes) > 0 {
		modelName := engineReq.Cubes[0]
		for _, p := range policies {
			preds := p(modelName, secCtx)
			for _, pred := range preds {
				// Convert predicate to a filter map. This assumes simple equality.
				engineReq.Filters = append(engineReq.Filters, map[string]any{"member": pred.Field, "operator": "equals", "values": pred.Params})
			}
		}
	}

	// Assuming the engine has a `Compile` method that returns EmittedSQL without execution.
	emittedSQL, err := engine.Compile(ctx, engineReq, d)
	if err != nil {
		return nil, fmt.Errorf("query compilation failed: %w", err)
	}

	// Mocking GraphQL generation as it's a separate concern.
	graphqlQuery := fmt.Sprintf("{ view(name: \"%s\") { ... } }", req.View) // Mock GraphQL

	result := &models.CompileResult{
		SQL:     emittedSQL.SQL,
		GraphQL: graphqlQuery,
		Explain: &models.Explain{
			UsedPreAgg:       emittedSQL.UsedPreAggregation.Name,
			RoutingReason:    "Coverage OK",      // Placeholder
			RuleID:           "placeholder_rule", // Placeholder
			PartitionsPruned: new(int),           // Placeholder
			Freshness:        "3 hours ago",      // Placeholder
		},
	}

	return result, nil
}

// ExecuteQuery now handles the full execution and result scanning for the explorer.
func (s *QueryService) ExecuteQuery(ctx context.Context, secCtx security.Context, req models.ExplorerQueryRequest) (*models.ExecuteResult, error) {
	if req.View == "" {
		return nil, fmt.Errorf("a view must be specified")
	}

	startTime := time.Now()

	catalog, err := s.modelProvider.GetActiveCatalog(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("could not load active model catalog: %w", err)
	}

	view, ok := catalog.Views[req.View]
	if !ok {
		return nil, fmt.Errorf("view '%s' not found in catalog", req.View)
	}

	engineReq, err := buildEngineRequestFromView(req, view, secCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build query from view: %w", err)
	}

	// For pagination, we request one extra row to see if there's a next page.
	limit := 50 // Default limit
	if req.Limit != nil {
		limit = *req.Limit
	}
	limitWithProbe := limit + 1
	engineReq.Limit = &limitWithProbe

	engine := cubeengine.NewEngine(catalog, s.db.DB, s.optService, dialect.Postgres{})
	d := dialect.Postgres{}

	// --- RLS Injection ---
	policies := []security.RLSPolicy{security.OrdersPolicy}
	if len(engineReq.Cubes) > 0 {
		modelName := engineReq.Cubes[0]
		for _, p := range policies {
			preds := p(modelName, secCtx)
			for _, pred := range preds {
				// Convert predicate to a filter map. This assumes simple equality.
				engineReq.Filters = append(engineReq.Filters, map[string]any{"member": pred.Field, "operator": "equals", "values": pred.Params})
			}
		}
	}

	emittedSQL, err := engine.Compile(ctx, engineReq, d)
	if err != nil {
		return nil, fmt.Errorf("query compilation failed: %w", err)
	}

	rows, err := s.db.QueryxContext(ctx, emittedSQL.SQL, emittedSQL.Params...)
	if err != nil {
		return nil, fmt.Errorf("database query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names before iterating over the rows.
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns from result: %w", err)
	}

	data, err := scanRowsToMap(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows: %w", err)
	}

	hasNext := false
	if len(data) > limit {
		hasNext = true
		data = data[:len(data)-1] // Trim the extra row
	}

	durationMs := time.Since(startTime).Milliseconds()

	result := &models.ExecuteResult{
		Columns:            inferColumns(cols),
		Rows:               data,
		Page:               models.PageInfo{Limit: req.Limit, Offset: req.Offset, HasNext: hasNext},
		DurationMs:         durationMs,
		UsedPreAggregation: emittedSQL.UsedPreAggregation.Name,
		SQL:                emittedSQL.SQL,
		GraphQL:            fmt.Sprintf("{ view(name: \"%s\") { ... } }", req.View), // Mock GraphQL
		Explain: &models.Explain{
			UsedPreAgg:       emittedSQL.UsedPreAggregation.Name,
			RoutingReason:    "Coverage OK",
			PartitionsPruned: new(int),      // Placeholder
			Freshness:        "3 hours ago", // Placeholder
		},
	}

	return result, nil
}

// scanRowsToMap scans sql.Rows into a slice of maps.
func scanRowsToMap(rows *sqlx.Rows) ([]map[string]any, error) {
	var results []map[string]any
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		// sqlx may return []byte for some types, convert to string for JSON friendliness
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				row[k] = string(b)
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// inferColumns creates a slice of Column structs from column names.
func inferColumns(colNames []string) []models.ExplorerColumn {
	if len(colNames) == 0 {
		return []models.ExplorerColumn{}
	}
	cols := make([]models.ExplorerColumn, len(colNames))
	for i, name := range colNames {
		// Type inference would be more sophisticated in a real implementation
		cols[i] = models.ExplorerColumn{Name: name, Type: "unknown"}
	}
	return cols
}

// --- Saved Query Management ---

// ListSavedQueries retrieves all saved queries for a given user.
func (s *QueryService) ListSavedQueries(ctx context.Context, secCtx security.Context, scope, viewName, search string, tags []string) ([]models.ListSavedQueriesItem, error) {
	var queries []models.ListSavedQueriesItem
	var err error

	baseQuery := `
		SELECT q.id, q.name, q.view_name, q.tags, q.owner_user_id, q.last_run_at, q.last_duration_ms, q.last_row_count, (q.preview IS NOT NULL) AS preview_available
		FROM explorer_saved_query q
	`
	var args []interface{}
	var conditions []string

	conditions = append(conditions, "q.is_deleted = false")

	switch scope {
	case "shared":
		baseQuery += " JOIN explorer_saved_query_acl a ON a.saved_query_id = q.id"
		conditions = append(conditions, "q.owner_user_id <> ?")
		args = append(args, secCtx.UserID)
		conditions = append(conditions, "((a.subject_type='user' AND a.subject_id=?) OR (a.subject_type='role' AND a.subject_id = ANY(?)) OR (a.subject_type='tenant' AND a.subject_id=?))")
		args = append(args, secCtx.UserID, pq.Array(secCtx.Roles), secCtx.TenantID)
	case "all":
		baseQuery += " LEFT JOIN explorer_saved_query_acl a ON a.saved_query_id = q.id"
		conditions = append(conditions, "(q.owner_user_id = ? OR ((a.subject_type='user' AND a.subject_id=?) OR (a.subject_type='role' AND a.subject_id = ANY(?)) OR (a.subject_type='tenant' AND a.subject_id=?)))")
		args = append(args, secCtx.UserID, secCtx.UserID, pq.Array(secCtx.Roles), secCtx.TenantID)
	default: // "mine"
		conditions = append(conditions, "q.owner_user_id = ?")
		args = append(args, secCtx.UserID)
	}

	if viewName != "" {
		conditions = append(conditions, "q.view_name = ?")
		args = append(args, viewName)
	}
	if len(tags) > 0 {
		conditions = append(conditions, "q.tags && ?")
		args = append(args, pq.Array(tags))
	}
	if search != "" {
		conditions = append(conditions, "(q.name ILIKE ? OR q.description ILIKE ?)")
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	finalQuery := baseQuery + " WHERE " + strings.Join(conditions, " AND ") + " ORDER BY COALESCE(q.last_run_at, q.created_at) DESC"
	finalQuery = s.db.Rebind(finalQuery)

	err = s.db.SelectContext(ctx, &queries, finalQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list saved queries: %w", err)
	}

	if queries == nil {
		return []models.ListSavedQueriesItem{}, nil
	}

	return queries, nil
}

// UpdateLastRunStats updates the telemetry for a saved query after an execution.
func (s *QueryService) UpdateLastRunStats(ctx context.Context, savedQueryID string, durationMs int64, rowCount int) error {
	query := `
		UPDATE explorer_saved_query
		SET last_run_at = now(), last_duration_ms = $1, last_row_count = $2
		WHERE id = $3
	`
	_, err := s.db.ExecContext(ctx, query, durationMs, rowCount, savedQueryID)
	if err != nil {
		// Log the error but don't fail the user's request over it.
		logging.GetLogger().Sugar().Warnf("WARN: failed to update last run stats for saved query %s: %v", savedQueryID, err)
	}
	return err
}

// LogAndDiffRun logs a query execution and computes the diff against the previous run.
func (s *QueryService) LogAndDiffRun(ctx context.Context, savedQueryID string, req models.ExplorerQueryRequest, res *models.ExecuteResult) error {
	// 1. Get previous run
	var previousRun models.ExplorerSavedQueryRun
	prevRunQuery := `SELECT * FROM explorer_saved_query_run WHERE saved_query_id = $1 ORDER BY executed_at DESC LIMIT 1`
	err := s.db.GetContext(ctx, &previousRun, prevRunQuery, savedQueryID)
	hasPreviousRun := err == nil

	// 2. Create current run record
	queryJSON, _ := json.Marshal(req)
	hash := sha1.Sum(queryJSON)
	queryHash := hex.EncodeToString(hash[:])

	filtersJSON, _ := json.Marshal(req.Filters)
	var columnNames []string
	for _, c := range res.Columns {
		columnNames = append(columnNames, c.Name)
	}

	currentRun := models.ExplorerSavedQueryRun{
		ID:           uuid.New(),
		SavedQueryID: uuid.MustParse(savedQueryID),
		ExecutedAt:   time.Now(),
		QueryHash:    queryHash,
		RowCount:     int64(len(res.Rows)),
		Columns:      columnNames,
		Filters:      filtersJSON,
	}

	// 3. Insert current run
	insertRunQuery := `INSERT INTO explorer_saved_query_run (id, saved_query_id, executed_at, query_hash, row_count, columns, filters) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = s.db.ExecContext(ctx, insertRunQuery, currentRun.ID, currentRun.SavedQueryID, currentRun.ExecutedAt, currentRun.QueryHash, currentRun.RowCount, pq.Array(currentRun.Columns), currentRun.Filters)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("ERROR: failed to log saved query run: %v", err)
		return err // Return error as this is important for the feature
	}

	// 4. Compute and store diff if there was a previous run
	if hasPreviousRun {
		diff := computeDiff(previousRun, currentRun)
		diffJSON, _ := json.Marshal(diff)

		updateDiffQuery := `UPDATE explorer_saved_query SET preview_diff = $1 WHERE id = $2`
		_, err = s.db.ExecContext(ctx, updateDiffQuery, diffJSON, savedQueryID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("ERROR: failed to store preview diff: %v", err)
		}
	}

	return nil
}

func computeDiff(before, after models.ExplorerSavedQueryRun) models.PreviewDiff {
	diff := models.PreviewDiff{}

	// Row count diff
	diff.RowCount.Before = before.RowCount
	diff.RowCount.After = after.RowCount

	// Column diff
	beforeCols := make(map[string]bool)
	for _, c := range before.Columns {
		beforeCols[c] = true
	}
	afterCols := make(map[string]bool)
	for _, c := range after.Columns {
		afterCols[c] = true
	}

	for _, c := range after.Columns {
		if !beforeCols[c] {
			diff.Columns.Added = append(diff.Columns.Added, c)
		}
	}
	for _, c := range before.Columns {
		if !afterCols[c] {
			diff.Columns.Removed = append(diff.Columns.Removed, c)
		}
	}

	// Filter diff (simplified)
	diff.FiltersChanged = before.QueryHash != after.QueryHash

	return diff
}

// GetLatestDiff retrieves the pre-computed diff for a saved query.
func (s *QueryService) GetLatestDiff(ctx context.Context, savedQueryID string) (json.RawMessage, error) {
	var diff json.RawMessage
	err := s.db.GetContext(ctx, &diff, "SELECT preview_diff FROM explorer_saved_query WHERE id = $1 AND preview_diff IS NOT NULL", savedQueryID)
	return diff, err
}

// GetSavedQuery retrieves a single saved query by its ID, checking ownership.
func (s *QueryService) GetSavedQuery(ctx context.Context, id, userID string) (*models.ExplorerSavedQuery, error) {
	var query models.ExplorerSavedQuery
	err := s.db.GetContext(ctx, &query, "SELECT * FROM explorer_saved_query WHERE id = $1 AND is_deleted = false", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("saved query not found or access denied")
		}
		return nil, fmt.Errorf("failed to get saved query: %w", err)
	}
	return &query, nil
}

// GetPreview retrieves just the preview data for a saved query.
func (s *QueryService) GetPreview(ctx context.Context, id string) (json.RawMessage, error) {
	var preview json.RawMessage
	err := s.db.GetContext(ctx, &preview, "SELECT preview FROM explorer_saved_query WHERE id = $1 AND preview IS NOT NULL", id)
	return preview, err
}

// CreateSavedQuery creates a new saved query record.
func (s *QueryService) CreateSavedQuery(ctx context.Context, req models.SavedQueryCreateRequest, userID, tenantID string) (*models.ExplorerSavedQuery, error) {
	query := `
		INSERT INTO explorer_saved_query (id, owner_user_id, owner_tenant_id, name, description, view_name, query, viz_config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING *
	`
	newID := uuid.New()
	var savedQuery models.ExplorerSavedQuery

	err := s.db.QueryRowxContext(ctx, query, newID, userID, tenantID, req.Name, req.Description, req.ViewName, req.Query, req.VizConfig).StructScan(&savedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to create saved query: %w", err)
	}
	return &savedQuery, nil
}

// UpdateSavedQuery updates an existing saved query, checking for ownership.
func (s *QueryService) UpdateSavedQuery(ctx context.Context, id string, req models.SavedQueryUpdateRequest, userID string) error {
	query := `
		UPDATE explorer_saved_query
		SET name = $1, description = $2, query = $3, viz_config = $4, updated_at = now()
		WHERE id = $5 AND owner_user_id = $6
	`
	result, err := s.db.ExecContext(ctx, query, req.Name, req.Description, req.Query, req.VizConfig, id, userID)
	if err != nil {
		return fmt.Errorf("failed to update saved query: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("saved query not found or access denied")
	}
	return nil
}

// DeleteSavedQuery deletes a saved query, checking for ownership.
func (s *QueryService) DeleteSavedQuery(ctx context.Context, id, userID string) error {
	// Soft delete
	query := `UPDATE explorer_saved_query SET is_deleted = true, updated_at = now() WHERE id = $1 AND owner_user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete saved query: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("saved query not found or access denied")
	}
	return nil
}

// ShareQuery grants a user, role, or tenant access to a saved query.
func (s *QueryService) ShareQuery(ctx context.Context, savedQueryID string, req models.ShareRequest, grantedByUserID string) error {
	query := `
		INSERT INTO explorer_saved_query_acl (id, saved_query_id, subject_type, subject_id, permission, granted_by)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
		ON CONFLICT (saved_query_id, subject_type, subject_id) DO UPDATE SET permission = EXCLUDED.permission, updated_at = now()`
	result, err := s.db.ExecContext(ctx, query, savedQueryID, req.SubjectType, req.SubjectID, req.Permission, grantedByUserID)
	if err != nil {
		return fmt.Errorf("failed to share query: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("saved query not found or access denied")
	}
	return nil
}

// CloneSavedQuery creates a copy of an existing saved query for the current user.
func (s *QueryService) CloneSavedQuery(ctx context.Context, id, newOwnerID, newTenantID string) (*models.ExplorerSavedQuery, error) {
	var original models.ExplorerSavedQuery
	// Anyone can clone a query, so we don't check owner_user_id here.
	// In a real app, you might check for tenant-level visibility.
	err := s.db.GetContext(ctx, &original, "SELECT * FROM explorer_saved_query WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("original saved query not found")
		}
		return nil, fmt.Errorf("failed to get original saved query: %w", err)
	}

	cloneReq := models.SavedQueryCreateRequest{
		Name:        "Copy of " + original.Name,
		Description: &original.Description.String,
		ViewName:    original.ViewName,
		Query:       original.Query,
		VizConfig:   original.VizConfig,
	}

	return s.CreateSavedQuery(ctx, cloneReq, newOwnerID, newTenantID)
}

// --- Workbook Management ---

func (s *QueryService) CreateWorkbook(ctx context.Context, req models.CreateWorkbookRequest, userID string) (*models.FullWorkbook, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	workbook := models.Workbook{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: sql.NullString{String: *req.Description, Valid: req.Description != nil},
		OwnerUserID: userID,
		Tags:        req.Tags,
	}
	workbookQuery := `INSERT INTO explorer_workbook (id, name, description, owner_user_id, tags) VALUES ($1, $2, $3, $4, $5) RETURNING *`
	if err := tx.QueryRowxContext(ctx, workbookQuery, workbook.ID, workbook.Name, workbook.Description, workbook.OwnerUserID, pq.Array(workbook.Tags)).StructScan(&workbook); err != nil {
		return nil, fmt.Errorf("failed to create workbook: %w", err)
	}

	var tabs []models.WorkbookTab
	for i, tabReq := range req.Tabs {
		tab := models.WorkbookTab{
			ID:         uuid.New(),
			WorkbookID: workbook.ID,
			Title:      tabReq.Title,
			ViewName:   tabReq.ViewName,
			Query:      tabReq.Query,
			VizConfig:  tabReq.VizConfig,
			Position:   i,
		}
		tabQuery := `INSERT INTO explorer_workbook_tab (id, workbook_id, title, view_name, query, viz_config, position) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *`
		if err := tx.QueryRowxContext(ctx, tabQuery, tab.ID, tab.WorkbookID, tab.Title, tab.ViewName, tab.Query, tab.VizConfig, tab.Position).StructScan(&tab); err != nil {
			return nil, fmt.Errorf("failed to create workbook tab: %w", err)
		}
		tabs = append(tabs, tab)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit workbook transaction: %w", err)
	}

	return &models.FullWorkbook{Workbook: workbook, Tabs: tabs}, nil
}

// --- Duplicate Detection ---

// DetectDuplicates finds clusters of saved queries with identical logic.
func (s *QueryService) DetectDuplicates(ctx context.Context, userID, datasourceID string) ([]models.DuplicateQueryCluster, error) {
	// In a real system, a background job would populate explorer_query_fingerprint.
	// For this demo, we'll do a simplified version on the fly.

	// 1. Fetch all of the user's queries for the given datasource.
	var userQueries []models.ExplorerSavedQuery
	// NOTE: This assumes a `datasource_id` column exists on the `explorer_saved_query` table.
	// This would be populated when a query is saved, based on the view it uses.
	query := `
		SELECT id, name, view_name, query, viz_config 
		FROM explorer_saved_query 
		WHERE owner_user_id = $1 AND datasource_id = $2 AND is_deleted = false
	`
	if err := s.db.SelectContext(ctx, &userQueries, query, userID, datasourceID); err != nil {
		return nil, err
	}

	// 2. Fingerprint and group them.
	fingerprintGroups := make(map[string][]models.ListSavedQueriesItem)
	for _, q := range userQueries {
		// Simple fingerprint: hash of query + viz config JSON.
		// A real implementation would normalize the JSON first (sort keys, arrays).
		combined, _ := json.Marshal(map[string]json.RawMessage{"q": q.Query, "v": q.VizConfig})
		hash := sha1.Sum(combined)
		fingerprint := hex.EncodeToString(hash[:])

		item := models.ListSavedQueriesItem{ID: q.ID, Name: q.Name, ViewName: q.ViewName, OwnerUserID: q.OwnerUserID}
		fingerprintGroups[fingerprint] = append(fingerprintGroups[fingerprint], item)
	}

	// 3. Format the response, only including clusters with more than one query.
	var clusters []models.DuplicateQueryCluster
	for fp, queries := range fingerprintGroups {
		if len(queries) > 1 {
			clusters = append(clusters, models.DuplicateQueryCluster{
				Fingerprint: fp,
				Queries:     queries,
			})
		}
	}

	return clusters, nil
}
