package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	Dimensions []SavedQueryDimension     `json:"dimensions"`
	Measures   []SavedQueryMeasure       `json:"measures"`
	Filters    []SavedQueryFilter        `json:"filters"`
	Parameters []SavedQueryParameter     `json:"parameters"`
	Limit      int                       `json:"limit,omitempty"`
	// Subject pins the analytical subject (business_object | cube) for execute.
	// Cube rows also set source_kind='cube' / source_id=cubeId on the table row.
	Subject *boresolver.QuerySubject `json:"subject,omitempty"`
}

const (
	savedQuerySourceBusinessObject = "business_object"
	savedQuerySourceCube           = "cube"
)

// SavedQuery is one row of data_explorer.saved_query, with QueryState
// decoded for JSON responses instead of the raw jsonb bytes.
type SavedQuery struct {
	ID            string              `json:"id"`
	TenantID      string              `json:"tenantId"`
	UserID        string              `json:"userId"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	// SourceKind is business_object (default) or cube. For cube rows, BOID
	// carries the cube id (source_id) for backward-compatible list filters.
	SourceKind    string                   `json:"sourceKind,omitempty"`
	BOID          string                   `json:"boId"`
	BindingID     string                   `json:"bindingId"`
	RelatedBOIDs  []string                 `json:"relatedBoIds"`
	Subject       *boresolver.QuerySubject `json:"subject,omitempty"`
	ChartType     string                   `json:"chartType"`
	State         SavedQueryState          `json:"state"`
	Tags          []string                 `json:"tags"`
	FolderID      string                   `json:"folderId,omitempty"`
	IsFavorite    bool                     `json:"isFavorite"`
	Visibility    string                   `json:"visibility"`
	IsCore        bool                     `json:"isCore"`
	Status        string                   `json:"status,omitempty"` // active | deprecated | archived
	ArchivedAt    *time.Time               `json:"archivedAt,omitempty"`
	CreatedBy     string                   `json:"createdBy,omitempty"`
	CreatedAt     time.Time                `json:"createdAt"`
	UpdatedAt     time.Time                `json:"updatedAt"`
	CoreStatus    string                   `json:"coreStatus,omitempty"` // core | vanilla | extended | upgrade_available | cloned | custom
	Customization *QueryCustomization      `json:"customization,omitempty"`
	Editable      bool                     `json:"editable"`
	CanCustomize  bool                     `json:"canCustomize"`
	ClonedFrom    *QueryCloneSource        `json:"clonedFrom,omitempty"`
}

type savedQueryRow struct {
	ID           string         `db:"id"`
	TenantID     string         `db:"tenant_id"`
	UserID       string         `db:"user_id"`
	Name         string         `db:"name"`
	Description  string         `db:"description"`
	SourceKind   string         `db:"source_kind"`
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
	Status       string         `db:"status"`
	ArchivedAt   *time.Time     `db:"archived_at"`
	CreatedBy    sql.NullString `db:"created_by"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

const savedQuerySelectCols = `id, tenant_id, user_id, name, description, COALESCE(source_kind, 'business_object') AS source_kind, source_id, binding_id, related_bo_ids,
	chart_type, query_state, tags, folder_id, is_favorite, visibility, is_core, COALESCE(status, 'active') AS status, archived_at, created_by, created_at, updated_at`

func (r savedQueryRow) toSavedQuery() SavedQuery {
	var state SavedQueryState
	_ = json.Unmarshal(r.QueryState, &state)
	kind := r.SourceKind
	if kind == "" {
		kind = savedQuerySourceBusinessObject
	}
	sq := SavedQuery{
		ID:           r.ID,
		TenantID:     r.TenantID,
		UserID:       r.UserID,
		Name:         r.Name,
		Description:  r.Description,
		SourceKind:   kind,
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
		Status:       r.Status,
		ArchivedAt:   r.ArchivedAt,
		CreatedBy:    r.CreatedBy.String,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		Editable:     true,
		CanCustomize: false,
		CoreStatus:   "custom",
	}
	sq.Subject = resolvedSavedQuerySubject(sq)
	return sq
}

// resolvedSavedQuerySubject returns the explicit subject from query_state, or
// synthesizes one from source_kind / source_id for older rows.
func resolvedSavedQuerySubject(sq SavedQuery) *boresolver.QuerySubject {
	if sq.State.Subject != nil {
		return sq.State.Subject
	}
	if sq.SourceKind == savedQuerySourceCube {
		return &boresolver.QuerySubject{
			Kind:            boresolver.QuerySubjectCube,
			CubeID:          sq.BOID,
			ContractVersion: boresolver.ContractVersionPin{Latest: true},
		}
	}
	if sq.BOID == "" && sq.BindingID == "" {
		return nil
	}
	return &boresolver.QuerySubject{
		Kind:         boresolver.QuerySubjectBusinessObject,
		BOID:         sq.BOID,
		BindingID:    sq.BindingID,
		RelatedBOIDs: sq.RelatedBOIDs,
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
// HandleListSavedQueries handles GET /api/explorer/saved-queries?boId=...
// Returns the caller's tenant queries plus active core queries from the
// master tenant, decorated with core adoption status (vanilla, extended,
// upgrade_available, cloned).
func (h *SavedQueryHandler) HandleListSavedQueries(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	boID := r.URL.Query().Get("boId")
	folderID := r.URL.Query().Get("folderId")
	gold := h.goldCopyID(r.Context())
	isMaster := gold != "" && gold == secCtx.TenantID
	canCust := h.canCustomize(r, secCtx.TenantID)

	// 1. Fetch tenant-owned queries
	query := `SELECT ` + savedQuerySelectCols + `
	          FROM data_explorer.saved_query
	          WHERE tenant_id = $1 AND (visibility = 'shared' OR user_id = $2) AND archived_at IS NULL`
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
		sq := r.toSavedQuery()
		if sq.IsCore && isMaster {
			sq.CoreStatus = "core"
			sq.Editable = true
			sq.CanCustomize = false
		}
		out = append(out, sq)
	}

	// 2. If not the master tenant, also read-through core queries from master tenant
	if !isMaster && gold != "" {
		adoptionsMap, _ := h.adoptions(r.Context(), secCtx.TenantID)
		coreQuery := `SELECT ` + savedQuerySelectCols + `
		              FROM data_explorer.saved_query
		              WHERE tenant_id = $1 AND is_core = true AND archived_at IS NULL`
		coreArgs := []interface{}{gold}
		if boID != "" {
			coreArgs = append(coreArgs, boID)
			coreQuery += fmt.Sprintf(" AND source_id = $%d", len(coreArgs))
		}
		coreQuery += " ORDER BY name ASC"

		var coreRows []savedQueryRow
		if err := h.db.Select(&coreRows, coreQuery, coreArgs...); err == nil {
			for _, cr := range coreRows {
				csq := cr.toSavedQuery()
				adopt, hasAdopt := adoptionsMap[csq.ID]
				var adoptPtr *queryAdoption
				if hasAdopt {
					adoptPtr = &adopt
				}

				// If adopted as cloned, suppress the core entry because the clone already appears in the tenant list
				if adoptPtr != nil && adoptPtr.Mode == "cloned" {
					continue
				}

				// If deactivated in this tenant, skip
				if adoptPtr != nil && !adoptPtr.Active {
					continue
				}

				presentCoreQuery(&csq, adoptPtr, false, canCust)
				out = append(out, csq)
			}
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"savedQueries": out})
}

type savedQueryCreateRequest struct {
	Name         string                   `json:"name"`
	Description  string                   `json:"description"`
	BOID         string                   `json:"boId"`
	BindingID    string                   `json:"bindingId"`
	RelatedBOIDs []string                 `json:"relatedBoIds"`
	Subject      *boresolver.QuerySubject `json:"subject,omitempty"`
	ChartType    string                   `json:"chartType"`
	State        SavedQueryState          `json:"state"`
	Tags         []string                 `json:"tags"`
	FolderID     string                   `json:"folderId"`
}

// normalizeSavedQueryIdentity locks source_kind/source_id from subject or BO fields.
// Cube: source_kind='cube', source_id=cubeId, binding_id null. BO: existing rules.
func normalizeSavedQueryIdentity(req *savedQueryCreateRequest) (sourceKind, sourceID string, bindingID interface{}, err error) {
	if req == nil {
		return "", "", nil, errors.New("missing request")
	}
	subj := req.Subject
	if subj == nil {
		subj = req.State.Subject
	}
	if subj != nil && subj.NormalizedKind() == boresolver.QuerySubjectCube {
		cubeID := strings.TrimSpace(subj.CubeID)
		if cubeID == "" {
			return "", "", nil, errors.New("subject.cubeId is required for cube saved queries")
		}
		subj.Kind = boresolver.QuerySubjectCube
		subj.CubeID = cubeID
		req.Subject = subj
		req.State.Subject = subj
		return savedQuerySourceCube, cubeID, nil, nil
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.BOID) == "" {
		return "", "", nil, errors.New("name and boId are required")
	}
	boSubj := &boresolver.QuerySubject{
		Kind:         boresolver.QuerySubjectBusinessObject,
		BOID:         req.BOID,
		BindingID:    req.BindingID,
		RelatedBOIDs: req.RelatedBOIDs,
	}
	if req.State.Subject == nil {
		req.State.Subject = boSubj
	}
	req.Subject = req.State.Subject
	var binding interface{}
	if req.BindingID != "" {
		binding = req.BindingID
	}
	return savedQuerySourceBusinessObject, req.BOID, binding, nil
}

// cubeVisibleToTenant reports whether cubeID is an active cube owned by the
// tenant or a gold-core cube readable by the tenant (parity with CubeHandler.getByID).
func (h *SavedQueryHandler) cubeVisibleToTenant(ctx context.Context, tenantID, cubeID string) (bool, error) {
	var n int
	err := h.db.GetContext(ctx, &n, `
		SELECT 1 FROM data_explorer.cube_definition
		WHERE id = $1 AND tenant_id = $2 AND archived_at IS NULL
		LIMIT 1
	`, cubeID, tenantID)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	gold := h.goldCopyID(ctx)
	if gold == "" || gold == tenantID {
		return false, nil
	}
	err = h.db.GetContext(ctx, &n, `
		SELECT 1 FROM data_explorer.cube_definition
		WHERE id = $1 AND tenant_id = $2 AND is_core = true AND archived_at IS NULL
		LIMIT 1
	`, cubeID, gold)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
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
	if strings.TrimSpace(req.Name) == "" {
		h.writeError(w, errors.New("name is required"), http.StatusBadRequest)
		return
	}
	sourceKind, sourceID, bindingID, err := normalizeSavedQueryIdentity(&req)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	if sourceKind == savedQuerySourceCube {
		ok, err := h.cubeVisibleToTenant(r.Context(), secCtx.TenantID, sourceID)
		if err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		}
		if !ok {
			h.writeError(w, errors.New("forbidden: cube does not belong to the caller's tenant"), http.StatusForbidden)
			return
		}
	} else {
		if owned, err := h.service.BOBelongsToTenant(sourceID, secCtx.TenantID); err != nil {
			h.writeError(w, err, http.StatusInternalServerError)
			return
		} else if !owned {
			h.writeError(w, errors.New("forbidden: business object does not belong to the caller's tenant"), http.StatusForbidden)
			return
		}
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING `+savedQuerySelectCols+`
	`, id, secCtx.TenantID, secCtx.UserID, req.Name, req.Description, sourceKind, sourceID, bindingID, pq.Array(req.RelatedBOIDs), req.ChartType, stateBytes, pq.Array(req.Tags), folderID, secCtx.UserID).StructScan(&row)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to save query: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusCreated, row.toSavedQuery())
}

func (h *SavedQueryHandler) loadOwned(w http.ResponseWriter, r *http.Request) (*SavedQuery, string, bool) {
	secCtx, ctx, err := h.resolveAuth(r)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return nil, "", false
	}
	*r = *r.WithContext(ctx)
	id := chi.URLParam(r, "id")
	var row savedQueryRow
	err = h.db.Get(&row, `SELECT `+savedQuerySelectCols+`
	                      FROM data_explorer.saved_query
	                      WHERE id = $1 AND tenant_id = $2 AND (visibility = 'shared' OR user_id = $3)`,
		id, secCtx.TenantID, secCtx.UserID)
	if err == nil {
		sq := row.toSavedQuery()
		return &sq, secCtx.TenantID, true
	}

	// If not found in tenant's own queries, check if it's a core query from master tenant
	gold := h.goldCopyID(r.Context())
	if gold != "" && gold != secCtx.TenantID {
		var coreRow savedQueryRow
		err = h.db.Get(&coreRow, `SELECT `+savedQuerySelectCols+`
		                          FROM data_explorer.saved_query
		                          WHERE id = $1 AND is_core = true AND tenant_id = $2`,
			id, gold)
		if err == nil {
			sq := coreRow.toSavedQuery()
			a, _ := h.adoption(r.Context(), secCtx.TenantID, sq.ID)
			canCust := h.canCustomize(r, secCtx.TenantID)
			presentCoreQuery(&sq, a, false, canCust)
			return &sq, secCtx.TenantID, true
		}
	}

	h.writeError(w, fmt.Errorf("saved query not found: %w", err), http.StatusNotFound)
	return nil, "", false
}

// HandleGetSavedQuery handles GET /api/explorer/saved-queries/{id}.
func (h *SavedQueryHandler) HandleGetSavedQuery(w http.ResponseWriter, r *http.Request) {
	sq, _, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	h.writeJSON(w, http.StatusOK, sq)
}

// HandleUpdateSavedQuery handles PUT /api/explorer/saved-queries/{id}.
func (h *SavedQueryHandler) HandleUpdateSavedQuery(w http.ResponseWriter, r *http.Request) {
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	if sq.IsCore && !h.isGoldCopy(r.Context(), tenantID) {
		h.writeError(w, errors.New("cannot update core query directly; use /extend to customize"), http.StatusForbidden)
		return
	}
	var req savedQueryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = sq.Name
	}
	if req.ChartType == "" {
		req.ChartType = sq.ChartType
	}
	// Source kind / source_id stay locked after create. Refresh subject in
	// query_state from the request when provided; otherwise keep existing.
	if req.Subject != nil {
		req.State.Subject = req.Subject
	} else if req.State.Subject == nil && sq.State.Subject != nil {
		req.State.Subject = sq.State.Subject
	}
	if sq.SourceKind == savedQuerySourceCube {
		// Cube identity is immutable; force subject.kind/cubeId to match the row.
		pin := boresolver.ContractVersionPin{Latest: true}
		if req.State.Subject != nil {
			pin = req.State.Subject.ContractVersion
		}
		req.State.Subject = &boresolver.QuerySubject{
			Kind:            boresolver.QuerySubjectCube,
			CubeID:          sq.BOID,
			ContractVersion: pin,
		}
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
	related := req.RelatedBOIDs
	if sq.SourceKind == savedQuerySourceCube {
		related = nil
	}
	var out savedQueryRow
	err = h.db.QueryRowx(`
		UPDATE data_explorer.saved_query
		SET name = $1, description = $2, chart_type = $3, query_state = $4, tags = $5,
		    related_bo_ids = $6, folder_id = $7, updated_at = NOW()
		WHERE id = $8 AND tenant_id = $9
		RETURNING `+savedQuerySelectCols+`
	`, req.Name, req.Description, req.ChartType, stateBytes, pq.Array(req.Tags),
		pq.Array(related), folderID, sq.ID, tenantID).StructScan(&out)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to update saved query: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, out.toSavedQuery())
}

// HandleSetFavorite handles PUT /api/explorer/saved-queries/{id}/favorite.
func (h *SavedQueryHandler) HandleSetFavorite(w http.ResponseWriter, r *http.Request) {
	sq, tenantID, ok := h.loadOwned(w, r)
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
	`, req.IsFavorite, sq.ID, tenantID).StructScan(&out)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to update favorite: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, out.toSavedQuery())
}

// HandleDeleteSavedQuery handles DELETE /api/explorer/saved-queries/{id}.
// Enforces:
// 1. Core query protection (403 for non-gold-copy tenants)
// 2. Server-side usage scan before delete (409 Conflict with usage report if referenced)
// 3. Core query adoption block (409 Conflict if adoption rows exist; recommends deprecation)
// 4. Soft-delete via archived_at timestamp and status = 'archived'
func (h *SavedQueryHandler) HandleDeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	secCtx, _, _ := handlers.SecurityContextFromRequest(r, "", "", h.deps)

	// 1. Core query protection: client tenants cannot delete core query
	if sq.IsCore && !h.isGoldCopy(r.Context(), secCtx.TenantID) {
		h.writeError(w, errors.New("cannot delete core query: core queries are managed by master tenant"), http.StatusForbidden)
		return
	}

	// 2. Server-side usage scan across pages, other queries, and core adoptions
	report, err := ScanSavedQueryUsage(r.Context(), h.db, sq.ID, sq.IsCore, secCtx.TenantID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to check query references: %w", err), http.StatusInternalServerError)
		return
	}

	if report.InUse {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Conflict",
			"message":    "Cannot delete saved query: query is currently referenced",
			"inUse":      true,
			"references": report.References,
		})
		return
	}

	// 3. Soft-delete the query (sets archived_at and status = 'archived')
	_, err = h.db.ExecContext(r.Context(), `
		UPDATE data_explorer.saved_query
		SET archived_at = NOW(), status = 'archived', updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`, sq.ID, tenantID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to delete saved query: %w", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleCloneSavedQuery handles POST /api/explorer/saved-queries/{id}/clone.
func (h *SavedQueryHandler) HandleCloneSavedQuery(w http.ResponseWriter, r *http.Request) {
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	secCtx, _, _ := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	newID := uuid.NewString()
	stateBytes, _ := json.Marshal(sq.State)
	sourceKind := sq.SourceKind
	if sourceKind == "" {
		sourceKind = savedQuerySourceBusinessObject
	}
	var bindingID interface{}
	if sourceKind != savedQuerySourceCube && sq.BindingID != "" {
		bindingID = sq.BindingID
	}

	var out savedQueryRow
	err := h.db.QueryRowx(`
		INSERT INTO data_explorer.saved_query
			(id, tenant_id, user_id, name, description, source_kind, source_id, binding_id, related_bo_ids, chart_type, query_state, tags, is_core, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, false, $3)
		RETURNING `+savedQuerySelectCols+`
	`, newID, tenantID, secCtx.UserID, sq.Name+" (copy)", sq.Description, sourceKind, sq.BOID, bindingID, pq.Array(sq.RelatedBOIDs), sq.ChartType, stateBytes, pq.Array(sq.Tags)).StructScan(&out)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to clone saved query: %w", err), http.StatusInternalServerError)
		return
	}

	cloned := out.toSavedQuery()
	if sq.IsCore {
		cloned.ClonedFrom = &QueryCloneSource{
			QueryID: sq.ID,
			Name:    sq.Name,
			Version: 1,
		}
		// Record adoption mode = 'cloned'
		_, _ = h.db.ExecContext(r.Context(), `
			INSERT INTO core_object_adoption (tenant_id, object_type, core_object_id, active, mode, base_version, clone_object_id, updated_at, updated_by)
			VALUES ($1, $2, $3, true, 'cloned', 1, $4, NOW(), $5)
			ON CONFLICT (tenant_id, object_type, core_object_id) DO UPDATE SET
				mode = 'cloned', clone_object_id = EXCLUDED.clone_object_id, updated_at = NOW(), updated_by = EXCLUDED.updated_by
		`, tenantID, coreQueryObjectType, sq.ID, newID, secCtx.UserID)
	}

	h.writeJSON(w, http.StatusCreated, cloned)
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
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	filters, err := resolveParams(sq.State, r.URL.Query())
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	limit := sq.State.Limit
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	qd := savedQueryDef(*sq, tenantID, filters, limit)

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
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	if sq.UserID != secCtx.UserID {
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
	`, req.Visibility, sq.ID, tenantID).StructScan(&out)
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

// savedQueryDef is the query a saved query runs, with its parameters resolved.
// Cube subjects set Context.Subject so QueryService.Execute uses RoutePinned.
func savedQueryDef(sq SavedQuery, tenantID string, filters []boresolver.FilterDef, limit int) *boresolver.QueryDef {
	subj := resolvedSavedQuerySubject(sq)
	ctx := boresolver.QueryContext{
		BOID:         sq.BOID,
		BindingID:    sq.BindingID,
		TenantID:     tenantID,
		RelatedBOIDs: sq.RelatedBOIDs,
		Subject:      subj,
	}
	// For cube subjects, BOID on context is not a business object — clear it so
	// opportunistic BO routing does not misfire if Subject were ignored.
	if subj != nil && subj.NormalizedKind() == boresolver.QuerySubjectCube {
		ctx.BOID = ""
		ctx.BindingID = ""
		ctx.RelatedBOIDs = nil
	}
	qd := &boresolver.QueryDef{
		Context: ctx,
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
	return qd
}
