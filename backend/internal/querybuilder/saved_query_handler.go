package querybuilder

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// SavedQueryParameter is a named placeholder a filter can reference (via
// SavedQueryFilter.ParamRef) instead of a hardcoded literal value - resolved
// from the caller's request query-string at run time. This is what lets one
// saved query be re-run with a different value from a REST client, or have
// its filter driven by a Page Studio Slicer/selection instead of being
// frozen to whatever value was selected when it was saved.
type SavedQueryParameter struct {
	Name     string      `json:"name"`
	Label    string      `json:"label,omitempty"`
	Type     string      `json:"type,omitempty"` // string|number|date|boolean
	Default  interface{} `json:"default,omitempty"`
	Required bool        `json:"required,omitempty"`
}

// SavedQueryFilter mirrors boresolver.FilterDef but allows Value to be
// deferred to a named parameter instead of being a literal baked in at
// save time.
type SavedQueryFilter struct {
	TermNodeID string      `json:"termNodeId"`
	Operator   string      `json:"operator"`
	Value      interface{} `json:"value,omitempty"`
	ParamRef   string      `json:"paramRef,omitempty"`
	BOID       string      `json:"boId,omitempty"`
}

type SavedQueryDimension struct {
	TermNodeID string `json:"termNodeId"`
	Alias      string `json:"alias"`
	BOID       string `json:"boId,omitempty"`
}

type SavedQueryMeasure struct {
	TermNodeID  string `json:"termNodeId"`
	Alias       string `json:"alias"`
	Aggregation string `json:"agg"`
	BOID        string `json:"boId,omitempty"`
}

// SavedQueryState is the shape persisted in data_explorer.saved_query.query_state.
type SavedQueryState struct {
	Dimensions []SavedQueryDimension `json:"dimensions"`
	Measures   []SavedQueryMeasure   `json:"measures"`
	Filters    []SavedQueryFilter    `json:"filters"`
	Parameters []SavedQueryParameter `json:"parameters"`
	Limit      int                   `json:"limit,omitempty"`
}

// SavedQuery is one row of data_explorer.saved_query, with QueryState
// decoded for JSON responses instead of the raw jsonb bytes.
type SavedQuery struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenantId"`
	UserID       string          `json:"userId"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	BOID         string          `json:"boId"`
	BindingID    string          `json:"bindingId"`
	RelatedBOIDs []string        `json:"relatedBoIds"`
	ChartType    string          `json:"chartType"`
	State        SavedQueryState `json:"state"`
	Tags         []string        `json:"tags"`
	FolderID     string          `json:"folderId,omitempty"`
	IsFavorite   bool            `json:"isFavorite"`
	Visibility   string          `json:"visibility"`
	IsCore       bool            `json:"isCore"`
	CreatedBy    string          `json:"createdBy,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type savedQueryRow struct {
	ID           string         `db:"id"`
	TenantID     string         `db:"tenant_id"`
	UserID       string         `db:"user_id"`
	Name         string         `db:"name"`
	Description  string         `db:"description"`
	SourceID     string         `db:"source_id"`
	BindingID    sql.NullString `db:"binding_id"`
	RelatedBOIDs pq.StringArray `db:"related_bo_ids"`
	ChartType    string         `db:"chart_type"`
	QueryState   []byte         `db:"query_state"`
	Tags         pq.StringArray `db:"tags"`
	FolderID     sql.NullString `db:"folder_id"`
	IsFavorite   bool           `db:"is_favorite"`
	Visibility   string         `db:"visibility"`
	IsCore       bool           `db:"is_core"`
	CreatedBy    sql.NullString `db:"created_by"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

const savedQuerySelectCols = `id, tenant_id, user_id, name, description, source_id, binding_id, related_bo_ids,
	chart_type, query_state, tags, folder_id, is_favorite, visibility, is_core, created_by, created_at, updated_at`

func (r savedQueryRow) toSavedQuery() SavedQuery {
	var state SavedQueryState
	_ = json.Unmarshal(r.QueryState, &state)
	return SavedQuery{
		ID:           r.ID,
		TenantID:     r.TenantID,
		UserID:       r.UserID,
		Name:         r.Name,
		Description:  r.Description,
		BOID:         r.SourceID,
		BindingID:    r.BindingID.String,
		RelatedBOIDs: []string(r.RelatedBOIDs),
		ChartType:    r.ChartType,
		State:        state,
		Tags:         []string(r.Tags),
		FolderID:     r.FolderID.String,
		IsFavorite:   r.IsFavorite,
		Visibility:   r.Visibility,
		IsCore:       r.IsCore,
		CreatedBy:    r.CreatedBy.String,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// SavedQueryHandler implements the /api/explorer/saved-queries endpoints
// (previously wired to no-op stubs, see git history of query_stubs.go) -
// persists a reusable, parameterized QueryDef per Business Object and
// re-executes it through the exact same QueryService/Executor pipeline
// every Page Studio widget already uses, so tenant scoping and SQL
// generation behave identically whether a query is ad-hoc or saved.
//
// Lives in package querybuilder (not internal/handlers, where the old
// stub lived) because it depends on *QueryService/Executor, and
// querybuilder already depends on internal/handlers (for
// SecurityContextDeps) - handlers depending back on querybuilder would be
// an import cycle.
type SavedQueryHandler struct {
	db       *sqlx.DB
	service  *QueryService
	executor Executor
	deps     handlers.SecurityContextDeps
}

func NewSavedQueryHandler(db *sqlx.DB, service *QueryService, executor Executor, deps handlers.SecurityContextDeps) *SavedQueryHandler {
	return &SavedQueryHandler{db: db, service: service, executor: executor, deps: deps}
}

func (h *SavedQueryHandler) writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logging.GetLogger().Sugar().Errorf("saved-query: failed to encode response: %v", err)
	}
}

func (h *SavedQueryHandler) writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   http.StatusText(status),
		"details": err.Error(),
	})
}

// HandleListSavedQueries handles GET /api/explorer/saved-queries?boId=...
// Scoped to the caller's tenant; boId narrows to queries built against one
// Business Object - the shape a Page Studio Chart/Slicer/KPI widget's
// "use a saved query" picker needs.
func (h *SavedQueryHandler) HandleListSavedQueries(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	boID := r.URL.Query().Get("boId")
	folderID := r.URL.Query().Get("folderId")
	// Visibility: a caller sees their own private queries plus every
	// 'shared' query in the tenant (report-builder's is_public/is_personal
	// pattern, collapsed to one flag - see the visibility-model decision
	// in this feature's planning).
	query := `SELECT ` + savedQuerySelectCols + `
	          FROM data_explorer.saved_query
	          WHERE tenant_id = $1 AND (visibility = 'shared' OR user_id = $2)`
	args := []interface{}{secCtx.TenantID, secCtx.UserID}
	if boID != "" {
		args = append(args, boID)
		query += fmt.Sprintf(" AND source_id = $%d", len(args))
	}
	if folderID != "" {
		args = append(args, folderID)
		query += fmt.Sprintf(" AND folder_id = $%d", len(args))
	}
	query += " ORDER BY is_favorite DESC, updated_at DESC"

	var rows []savedQueryRow
	if err := h.db.Select(&rows, query, args...); err != nil {
		h.writeError(w, fmt.Errorf("failed to list saved queries: %w", err), http.StatusInternalServerError)
		return
	}
	out := make([]SavedQuery, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toSavedQuery())
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"savedQueries": out})
}

type savedQueryCreateRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	BOID         string          `json:"boId"`
	BindingID    string          `json:"bindingId"`
	RelatedBOIDs []string        `json:"relatedBoIds"`
	ChartType    string          `json:"chartType"`
	State        SavedQueryState `json:"state"`
	Tags         []string        `json:"tags"`
	FolderID     string          `json:"folderId"`
}

// HandleCreateSavedQuery handles POST /api/explorer/saved-queries.
func (h *SavedQueryHandler) HandleCreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	var req savedQueryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.BOID == "" {
		h.writeError(w, errors.New("name and boId are required"), http.StatusBadRequest)
		return
	}
	if owned, err := h.service.BOBelongsToTenant(req.BOID, secCtx.TenantID); err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	} else if !owned {
		h.writeError(w, errors.New("forbidden: business object does not belong to the caller's tenant"), http.StatusForbidden)
		return
	}
	if req.ChartType == "" {
		req.ChartType = "bar"
	}

	stateBytes, err := json.Marshal(req.State)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	var folderID interface{}
	if req.FolderID != "" {
		folderID = req.FolderID
	}

	id := uuid.NewString()
	var row savedQueryRow
	err = h.db.QueryRowx(`
		INSERT INTO data_explorer.saved_query
			(id, tenant_id, user_id, name, description, source_kind, source_id, binding_id, related_bo_ids, chart_type, query_state, tags, folder_id, created_by)
		VALUES ($1, $2, $3, $4, $5, 'business_object', $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+savedQuerySelectCols+`
	`, id, secCtx.TenantID, secCtx.UserID, req.Name, req.Description, req.BOID, req.BindingID, pq.Array(req.RelatedBOIDs), req.ChartType, stateBytes, pq.Array(req.Tags), folderID, secCtx.UserID).StructScan(&row)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to save query: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusCreated, row.toSavedQuery())
}

func (h *SavedQueryHandler) loadOwned(w http.ResponseWriter, r *http.Request) (*savedQueryRow, string, bool) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return nil, "", false
	}
	id := chi.URLParam(r, "id")
	var row savedQueryRow
	err = h.db.Get(&row, `SELECT `+savedQuerySelectCols+`
	                      FROM data_explorer.saved_query
	                      WHERE id = $1 AND tenant_id = $2 AND (visibility = 'shared' OR user_id = $3)`,
		id, secCtx.TenantID, secCtx.UserID)
	if err != nil {
		h.writeError(w, fmt.Errorf("saved query not found: %w", err), http.StatusNotFound)
		return nil, "", false
	}
	return &row, secCtx.TenantID, true
}

// HandleGetSavedQuery handles GET /api/explorer/saved-queries/{id}.
func (h *SavedQueryHandler) HandleGetSavedQuery(w http.ResponseWriter, r *http.Request) {
	row, _, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	h.writeJSON(w, http.StatusOK, row.toSavedQuery())
}

// HandleUpdateSavedQuery handles PUT /api/explorer/saved-queries/{id}.
//
// The primary Business Object and its binding are locked at creation time
// (by design - "the primary BO and binding are static and cannot be
// changed" once a query exists, same as a report template's data source).
// Any boId/bindingId the request body carries is silently ignored rather
// than rejected, so a client that round-trips the full SavedQuery it just
// fetched (harmless) doesn't get a 400 for echoing back the same values.
// RelatedBOIDs, by contrast, is exactly what's meant to stay editable.
func (h *SavedQueryHandler) HandleUpdateSavedQuery(w http.ResponseWriter, r *http.Request) {
	row, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	var req savedQueryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = row.Name
	}
	if req.ChartType == "" {
		req.ChartType = row.ChartType
	}
	stateBytes, err := json.Marshal(req.State)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	var folderID interface{}
	if req.FolderID != "" {
		folderID = req.FolderID
	}
	var out savedQueryRow
	err = h.db.QueryRowx(`
		UPDATE data_explorer.saved_query
		SET name = $1, description = $2, chart_type = $3, query_state = $4, tags = $5,
		    related_bo_ids = $6, folder_id = $7, updated_at = NOW()
		WHERE id = $8 AND tenant_id = $9
		RETURNING `+savedQuerySelectCols+`
	`, req.Name, req.Description, req.ChartType, stateBytes, pq.Array(req.Tags),
		pq.Array(req.RelatedBOIDs), folderID, row.ID, tenantID).StructScan(&out)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to update saved query: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, out.toSavedQuery())
}

// HandleSetFavorite handles PUT /api/explorer/saved-queries/{id}/favorite,
// body {"isFavorite": true|false}. Favoriting is per-user, not per-query
// visibility, so it's a separate small mutation rather than folded into
// the general update (which a 'shared' query's non-owner can't call).
func (h *SavedQueryHandler) HandleSetFavorite(w http.ResponseWriter, r *http.Request) {
	row, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	var req struct {
		IsFavorite bool `json:"isFavorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	var out savedQueryRow
	err := h.db.QueryRowx(`
		UPDATE data_explorer.saved_query SET is_favorite = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
		RETURNING `+savedQuerySelectCols+`
	`, req.IsFavorite, row.ID, tenantID).StructScan(&out)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to update favorite: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, out.toSavedQuery())
}

// HandleDeleteSavedQuery handles DELETE /api/explorer/saved-queries/{id}.
func (h *SavedQueryHandler) HandleDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	_, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := h.db.Exec(`DELETE FROM data_explorer.saved_query WHERE id = $1 AND tenant_id = $2`, id, tenantID); err != nil {
		h.writeError(w, fmt.Errorf("failed to delete saved query: %w", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleCloneSavedQuery handles POST /api/explorer/saved-queries/{id}/clone.
func (h *SavedQueryHandler) HandleCloneSavedQuery(w http.ResponseWriter, r *http.Request) {
	row, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	secCtx, _, _ := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	newID := uuid.NewString()
	var out savedQueryRow
	err := h.db.QueryRowx(`
		INSERT INTO data_explorer.saved_query
			(id, tenant_id, user_id, name, description, source_kind, source_id, binding_id, related_bo_ids, chart_type, query_state, tags, folder_id, created_by)
		SELECT $1, $2, $3, name || ' (copy)', description, source_kind, source_id, binding_id, related_bo_ids, chart_type, query_state, tags, folder_id, $3
		FROM data_explorer.saved_query WHERE id = $4 AND tenant_id = $2
		RETURNING `+savedQuerySelectCols+`
	`, newID, tenantID, secCtx.UserID, row.ID).StructScan(&out)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to clone saved query: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusCreated, out.toSavedQuery())
}

// resolveParams merges request query-string values into the saved query's
// filters: a filter with ParamRef set reads its value from
// `?<paramName>=value`, falling back to the parameter's default, and is
// dropped from the executed query entirely if neither is present and the
// parameter isn't required (an optional, unset filter simply doesn't
// filter). A required parameter with no value/default is an error - the
// REST caller (or the Page Studio widget wiring the parameter to a page
// filter) must supply it.
func resolveParams(state SavedQueryState, values map[string][]string) ([]boresolver.FilterDef, error) {
	paramsByName := make(map[string]SavedQueryParameter, len(state.Parameters))
	for _, p := range state.Parameters {
		paramsByName[p.Name] = p
	}

	resolved := make([]boresolver.FilterDef, 0, len(state.Filters))
	for _, f := range state.Filters {
		fd := boresolver.FilterDef{TermNodeID: f.TermNodeID, Operator: f.Operator, Value: f.Value, BOID: f.BOID}
		if f.ParamRef == "" {
			resolved = append(resolved, fd)
			continue
		}
		param := paramsByName[f.ParamRef]
		if raw, ok := values[f.ParamRef]; ok && len(raw) > 0 && raw[0] != "" {
			if len(raw) > 1 || param.Type == "array" {
				fd.Value = raw
			} else {
				fd.Value = raw[0]
			}
			resolved = append(resolved, fd)
			continue
		}
		if param.Default != nil {
			fd.Value = param.Default
			resolved = append(resolved, fd)
			continue
		}
		if param.Required {
			return nil, fmt.Errorf("missing required parameter %q", f.ParamRef)
		}
		// Optional, unset, no default: drop this filter rather than send a
		// nil-valued predicate to the SQL generator.
	}
	return resolved, nil
}

// HandleGetPreview handles GET /api/explorer/saved-queries/{id}/preview -
// this doubles as the saved query's public REST data endpoint: run it with
// `?<paramName>=value` overrides (same auth as every other endpoint - a
// session JWT or an X-API-Key header, both handled transparently by
// AuthContextMiddleware) and get back real rows, not a stored sample.
func (h *SavedQueryHandler) HandleGetPreview(w http.ResponseWriter, r *http.Request) {
	row, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	sq := row.toSavedQuery()

	filters, err := resolveParams(sq.State, r.URL.Query())
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	limit := sq.State.Limit
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	qd := &boresolver.QueryDef{
		Context: boresolver.QueryContext{BOID: sq.BOID, BindingID: sq.BindingID, TenantID: tenantID, RelatedBOIDs: sq.RelatedBOIDs},
		Query: boresolver.QueryRequest{
			Dimensions: make([]boresolver.DimensionDef, len(sq.State.Dimensions)),
			Measures:   make([]boresolver.MeasureDef, len(sq.State.Measures)),
			Filters:    filters,
			Limit:      limit,
		},
	}
	for i, d := range sq.State.Dimensions {
		qd.Query.Dimensions[i] = boresolver.DimensionDef{TermNodeID: d.TermNodeID, Alias: d.Alias, BOID: d.BOID}
	}
	for i, m := range sq.State.Measures {
		qd.Query.Measures[i] = boresolver.MeasureDef{TermNodeID: m.TermNodeID, Alias: m.Alias, Aggregation: m.Aggregation, BOID: m.BOID}
	}

	db := h.executor.QueryDB(secCtx.DatasourceID)
	if db == nil {
		h.writeError(w, fmt.Errorf("no database connection for datasource %s", secCtx.DatasourceID), http.StatusInternalServerError)
		return
	}
	resp, err := h.service.Execute(ctx, secCtx, qd, db)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"columns":   resp.Columns,
		"rows":      resp.Rows,
		"rowCount":  resp.RowCount,
		"chartType": sq.ChartType,
		"name":      sq.Name,
		// hasRelatedBOs lets a consumer establish "this query traversed
		// zero relationships, so no join can have duplicated a row
		// relative to the primary BO" without per-column ownership/
		// cardinality metadata - which Preview's single-BO branch never
		// populates (Columns is empty by that branch's own contract).
		// That's the one fact a frontend roll-up-safety gate needs and
		// cannot derive itself: it has no access to this saved query's
		// RelatedBOIDs, only to what this endpoint returns. Computed
		// here, at the handler seam, from data already loaded (sq),
		// rather than adding it to QueryResultColumn or threading it
		// through the generator - it is a fact about the QUERY, not
		// about any column.
		"hasRelatedBOs": len(sq.RelatedBOIDs) > 0,
	})
}

// HandleGetDuplicates handles GET /api/explorer/saved-queries/duplicates -
// not a core part of this feature; returns an empty list rather than 404 so
// the frontend's (optional) "you already have a similar query" nudge has
// something well-formed to render, without pretending to do real duplicate
// detection.
func (h *SavedQueryHandler) HandleGetDuplicates(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"duplicates": []SavedQuery{}})
}

// HandleShareQuery handles POST /api/explorer/saved-queries/{id}/share,
// body {"visibility": "private"|"shared"} - toggles between private-to-owner
// and shared-with-the-whole-tenant (the visibility model chosen for this
// feature; a per-user/group ACL was considered and explicitly not built).
// Only the owner may change it - a non-owner can only ever have loaded this
// row via loadOwned's `visibility = 'shared'` branch, so row.UserID is
// checked explicitly rather than relying on that alone.
func (h *SavedQueryHandler) HandleShareQuery(w http.ResponseWriter, r *http.Request) {
	row, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	if row.UserID != secCtx.UserID {
		h.writeError(w, errors.New("only the owner can change sharing"), http.StatusForbidden)
		return
	}
	var req struct {
		Visibility string `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.Visibility != "private" && req.Visibility != "shared" {
		h.writeError(w, errors.New(`visibility must be "private" or "shared"`), http.StatusBadRequest)
		return
	}
	var out savedQueryRow
	err = h.db.QueryRowx(`
		UPDATE data_explorer.saved_query SET visibility = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
		RETURNING `+savedQuerySelectCols+`
	`, req.Visibility, row.ID, tenantID).StructScan(&out)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to update visibility: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, out.toSavedQuery())
}

// HandleGetDiff (version diffing) is out of scope for this pass - explicitly
// not implemented rather than faked, so a caller sees a clear "not built
// yet" instead of a silent no-op.

func (h *SavedQueryHandler) HandleGetDiff(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, errors.New("saved query version history is not implemented yet"), http.StatusNotImplemented)
}
