package querybuilder

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	coreCubeObjectType = "cube"
	cubeCursorVersion  = 1
	defaultCubeLimit   = 50
	maxCubeLimit       = 200
)

// CubeHandler serves /api/cubes CRUD, validate, and contract version publish.
//
// Tenant identity comes from auth only (SecurityContextFromRequest). List
// scopes mirror the table RLS/gold policy: own-tenant rows always, core gold
// rows when scope includes them. Adopted scope reads core_object_adoption
// with object_type = 'cube' (full adopt API lands in CUBE-4.2).
type CubeHandler struct {
	db   *sqlx.DB
	deps handlers.SecurityContextDeps
}

// NewCubeHandler constructs the cubes API handler.
func NewCubeHandler(db *sqlx.DB, deps handlers.SecurityContextDeps) *CubeHandler {
	return &CubeHandler{db: db, deps: deps}
}

const cubeDefSelectCols = `id, tenant_id, name, COALESCE(description,'') AS description, bo_id,
	dimensions, time_dimension, metric_ids, grains, materialization,
	COALESCE(federation, '{}'::jsonb) AS federation,
	COALESCE(contract_version, 1) AS contract_version,
	COALESCE(content_hash,'') AS content_hash, is_core, status, archived_at,
	created_by, created_at, updated_at`

func (h *CubeHandler) writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logging.GetLogger().Sugar().Errorf("cubes: failed to encode response: %v", err)
	}
}

func (h *CubeHandler) writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   http.StatusText(status),
		"details": err.Error(),
	})
}

func (h *CubeHandler) goldCopyID(ctx context.Context) string {
	tid := goldcopy.ResolveTenantID(ctx, h.db)
	if tid == uuid.Nil {
		return ""
	}
	return tid.String()
}

func (h *CubeHandler) isGoldCopy(ctx context.Context, tenantID string) bool {
	gold := h.goldCopyID(ctx)
	return gold != "" && gold == tenantID
}

type cubeCursor struct {
	Version   int       `json:"v"`
	UpdatedAt time.Time `json:"u"`
	ID        string    `json:"i"`
}

func encodeCubeCursor(c cubeCursor) string {
	c.Version = cubeCursorVersion
	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func decodeCubeCursor(s string) (cubeCursor, error) {
	if s == "" {
		return cubeCursor{}, fmt.Errorf("empty cursor")
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return cubeCursor{}, fmt.Errorf("invalid cursor")
	}
	var c cubeCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return cubeCursor{}, fmt.Errorf("invalid cursor")
	}
	if c.Version != cubeCursorVersion || c.ID == "" || c.UpdatedAt.IsZero() {
		return cubeCursor{}, fmt.Errorf("invalid cursor")
	}
	return c, nil
}

type cubeWriteRequest struct {
	Name            string                     `json:"name"`
	Description     string                     `json:"description"`
	BOID            string                     `json:"boId"`
	Dimensions      []CubeDimension            `json:"dimensions"`
	TimeDimension   *CubeTimeDimension         `json:"timeDimension,omitempty"`
	MetricIDs       []string                   `json:"metricIds"`
	Grains          [][]string                 `json:"grains"`
	Materialization *CubeMaterializationConfig `json:"materialization,omitempty"`
	Federation      *CubeFederation            `json:"federation,omitempty"`
	IsCore          *bool                      `json:"isCore,omitempty"`
	Status          string                     `json:"status,omitempty"`
	// FederationKeySamples optional orphan-gate fixtures (CUBE-2.1). When
	// federation is declared and samples are present, validate fails closed
	// if unmatched keys exceed orphanRateMaxPercent (default 1.0%).
	FederationKeySamples []FederationKeySample `json:"federationKeySamples,omitempty"`
}

func (req cubeWriteRequest) applyCreate(tenantID, userID string) CubeDefinition {
	c := CubeDefinition{
		TenantID:        tenantID,
		Name:            strings.TrimSpace(req.Name),
		Description:     req.Description,
		BOID:            strings.TrimSpace(req.BOID),
		Dimensions:      req.Dimensions,
		TimeDimension:   req.TimeDimension,
		MetricIDs:       req.MetricIDs,
		Grains:          req.Grains,
		ContractVersion: 1,
		Status:          "active",
		CreatedBy:       &userID,
	}
	if req.Materialization != nil {
		c.Materialization = *req.Materialization
	} else {
		c.Materialization = CubeMaterializationConfig{
			Strategy:    "starrocks_mv",
			HotEngine:   "starrocks",
			ColdEngine:  "iceberg",
			StalePolicy: "serve_with_flag",
		}
	}
	if req.Federation != nil {
		c.Federation = *req.Federation
	}
	if req.IsCore != nil {
		c.IsCore = *req.IsCore
	}
	if req.Status != "" {
		c.Status = req.Status
	}
	return c
}

func (req cubeWriteRequest) applyPatch(existing CubeDefinition) CubeDefinition {
	out := existing
	if req.Name != "" {
		out.Name = strings.TrimSpace(req.Name)
	}
	if req.BOID != "" {
		out.BOID = strings.TrimSpace(req.BOID)
	}
	if req.Dimensions != nil {
		out.Dimensions = req.Dimensions
	}
	if req.TimeDimension != nil {
		out.TimeDimension = req.TimeDimension
	}
	if req.MetricIDs != nil {
		out.MetricIDs = req.MetricIDs
	}
	if req.Grains != nil {
		out.Grains = req.Grains
	}
	if req.Materialization != nil {
		out.Materialization = *req.Materialization
	}
	if req.Federation != nil {
		out.Federation = *req.Federation
	}
	if req.Status != "" {
		out.Status = req.Status
	}
	return out
}

func bodyHasField(raw json.RawMessage, field string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[field]
	return ok
}

func (h *CubeHandler) loadMetrics(ctx context.Context, tenantID string, ids []string) (map[string]MetricDefinition, error) {
	out := make(map[string]MetricDefinition, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []metricDefRow
	err := h.db.SelectContext(ctx, &rows, `
		SELECT `+metricDefRowColumns+`
		FROM data_explorer.metric_definition
		WHERE (tenant_id = $1 OR is_core = true)
		  AND status = 'active' AND archived_at IS NULL
		  AND id::text = ANY($2)
	`, tenantID, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		m := row.toMetricDefinition()
		out[strings.ToLower(strings.TrimSpace(m.ID))] = m
	}
	return out, nil
}

func metricContentHashes(metrics map[string]MetricDefinition, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		norm := strings.ToLower(strings.TrimSpace(id))
		if m, ok := metrics[norm]; ok && m.ContentHash != "" {
			out = append(out, m.ContentHash)
		}
	}
	return out
}

func (h *CubeHandler) validateForWrite(ctx context.Context, tenantID string, c *CubeDefinition) error {
	if err := ValidateCubeStructural(*c); err != nil {
		return err
	}
	metrics, err := h.loadMetrics(ctx, tenantID, c.MetricIDs)
	if err != nil {
		return fmt.Errorf("load metrics: %w", err)
	}
	if err := ValidateCubeMetricReferences(*c, metrics); err != nil {
		return err
	}
	c.ContentHash = ComputeCubeContentHash(*c, metricContentHashes(metrics, c.MetricIDs))
	return nil
}

// GetCubeForTenant loads a cube visible to the tenant (own row or gold core).
// Used by deploy/refresh and the cube_refresh schedule runner (CUBE-1.5).
func (h *CubeHandler) GetCubeForTenant(ctx context.Context, tenantID, id string) (*CubeDefinition, error) {
	return h.getByID(ctx, tenantID, id)
}

// ListCubesForTenant lists cubes for schedule target pickers (CUBE-1.5).
func (h *CubeHandler) ListCubesForTenant(ctx context.Context, tenantID, scope, boID string, limit int, cursor string) ([]CubeDefinition, string, error) {
	if limit <= 0 {
		limit = defaultCubeLimit
	}
	gold := h.goldCopyID(ctx)
	var cur *cubeCursor
	if strings.TrimSpace(cursor) != "" {
		c, err := decodeCubeCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		cur = &c
	}
	items, next, err := h.listCubes(ctx, tenantID, gold, scope, boID, cur, limit)
	if err != nil {
		return nil, "", err
	}
	nextStr := ""
	if next != nil {
		nextStr = encodeCubeCursor(*next)
	}
	return items, nextStr, nil
}

func (h *CubeHandler) getByID(ctx context.Context, tenantID, id string) (*CubeDefinition, error) {
	var row cubeDefRow
	err := h.db.GetContext(ctx, &row, `
		SELECT `+cubeDefSelectCols+`
		FROM data_explorer.cube_definition
		WHERE id = $1 AND tenant_id = $2 AND archived_at IS NULL
	`, id, tenantID)
	if err == nil {
		c := row.toCubeDefinition()
		return &c, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Gold core read-through (RLS parity for non-RLS test paths / explicit filter).
	gold := h.goldCopyID(ctx)
	if gold == "" || gold == tenantID {
		return nil, sql.ErrNoRows
	}
	err = h.db.GetContext(ctx, &row, `
		SELECT `+cubeDefSelectCols+`
		FROM data_explorer.cube_definition
		WHERE id = $1 AND is_core = true AND tenant_id = $2 AND archived_at IS NULL
	`, id, gold)
	if err != nil {
		return nil, err
	}
	c := row.toCubeDefinition()
	return &c, nil
}

func (h *CubeHandler) insertCube(ctx context.Context, c CubeDefinition) (*CubeDefinition, error) {
	id := c.ID
	if id == "" {
		id = uuid.New().String()
	}
	dims, _ := json.Marshal(c.Dimensions)
	var td interface{}
	if c.TimeDimension != nil {
		b, _ := json.Marshal(c.TimeDimension)
		td = b
	}
	metrics, _ := json.Marshal(c.MetricIDs)
	grains, _ := json.Marshal(c.Grains)
	mat, _ := json.Marshal(c.Materialization)
	fed, _ := json.Marshal(c.Federation)
	if c.ContractVersion < 1 {
		c.ContractVersion = 1
	}
	if c.Status == "" {
		c.Status = "active"
	}

	var row cubeDefRow
	err := h.db.QueryRowxContext(ctx, `
		INSERT INTO data_explorer.cube_definition (
			id, tenant_id, name, description, bo_id,
			dimensions, time_dimension, metric_ids, grains, materialization,
			federation, contract_version, content_hash, is_core, status,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6::jsonb, $7::jsonb, $8::jsonb, $9::jsonb, $10::jsonb,
			$11::jsonb, $12, $13, $14, $15,
			$16, NOW(), NOW()
		)
		RETURNING `+cubeDefSelectCols, id, c.TenantID, c.Name, c.Description, c.BOID,
		dims, td, metrics, grains, mat, fed, c.ContractVersion, c.ContentHash, c.IsCore, c.Status, c.CreatedBy,
	).StructScan(&row)
	if err != nil {
		return nil, err
	}
	out := row.toCubeDefinition()
	return &out, nil
}

func (h *CubeHandler) updateCube(ctx context.Context, c CubeDefinition) (*CubeDefinition, error) {
	dims, _ := json.Marshal(c.Dimensions)
	var td interface{}
	if c.TimeDimension != nil {
		b, _ := json.Marshal(c.TimeDimension)
		td = b
	}
	metrics, _ := json.Marshal(c.MetricIDs)
	grains, _ := json.Marshal(c.Grains)
	mat, _ := json.Marshal(c.Materialization)
	fed, _ := json.Marshal(c.Federation)

	var archivedAt interface{}
	if c.Status == "archived" {
		now := time.Now().UTC()
		c.ArchivedAt = &now
		archivedAt = now
	} else {
		archivedAt = nil
		c.ArchivedAt = nil
	}

	var row cubeDefRow
	err := h.db.QueryRowxContext(ctx, `
		UPDATE data_explorer.cube_definition SET
			name = $1,
			description = $2,
			bo_id = $3,
			dimensions = $4::jsonb,
			time_dimension = $5::jsonb,
			metric_ids = $6::jsonb,
			grains = $7::jsonb,
			materialization = $8::jsonb,
			federation = $9::jsonb,
			contract_version = $10,
			content_hash = $11,
			status = $12,
			archived_at = $13,
			updated_at = NOW()
		WHERE id = $14 AND tenant_id = $15
		RETURNING `+cubeDefSelectCols,
		c.Name, c.Description, c.BOID, dims, td, metrics, grains, mat, fed,
		c.ContractVersion, c.ContentHash, c.Status, archivedAt, c.ID, c.TenantID,
	).StructScan(&row)
	if err != nil {
		return nil, err
	}
	out := row.toCubeDefinition()
	return &out, nil
}

// HandleListCubes handles GET /api/cubes?scope=core|custom|adopted|all&cursor=&limit=&boId=
func (h *CubeHandler) HandleListCubes(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	scope := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	if scope == "" {
		scope = "all"
	}
	switch scope {
	case "core", "custom", "adopted", "all":
	default:
		h.writeError(w, fmt.Errorf("scope must be core|custom|adopted|all"), http.StatusBadRequest)
		return
	}

	limit := defaultCubeLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			h.writeError(w, fmt.Errorf("limit must be a positive integer"), http.StatusBadRequest)
			return
		}
		if n > maxCubeLimit {
			n = maxCubeLimit
		}
		limit = n
	}

	boID := strings.TrimSpace(r.URL.Query().Get("boId"))
	var cur *cubeCursor
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		c, err := decodeCubeCursor(raw)
		if err != nil {
			h.writeError(w, err, http.StatusBadRequest)
			return
		}
		cur = &c
	}

	gold := h.goldCopyID(r.Context())
	cubes, next, err := h.listCubes(r.Context(), secCtx.TenantID, gold, scope, boID, cur, limit)
	if err != nil {
		h.writeError(w, fmt.Errorf("list cubes: %w", err), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"cubes": cubes,
		"scope": scope,
	}
	if next != nil {
		resp["nextCursor"] = encodeCubeCursor(*next)
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *CubeHandler) listCubes(
	ctx context.Context,
	tenantID, gold, scope, boID string,
	cur *cubeCursor,
	limit int,
) ([]CubeDefinition, *cubeCursor, error) {
	// Fetch limit+1 to detect a following page.
	fetch := limit + 1
	args := []interface{}{}
	where := []string{"archived_at IS NULL"}

	switch scope {
	case "custom":
		args = append(args, tenantID)
		where = append(where, fmt.Sprintf("tenant_id = $%d AND is_core = false", len(args)))
	case "core":
		if gold == "" {
			return []CubeDefinition{}, nil, nil
		}
		args = append(args, gold)
		where = append(where, fmt.Sprintf("tenant_id = $%d AND is_core = true", len(args)))
	case "adopted":
		args = append(args, tenantID, coreCubeObjectType)
		where = append(where, fmt.Sprintf(`
			id IN (
				SELECT core_object_id::uuid FROM public.core_object_adoption
				WHERE tenant_id = $%d AND object_type = $%d AND active = true
			)`, len(args)-1, len(args)))
	default: // all
		if gold != "" && gold != tenantID {
			args = append(args, tenantID, gold)
			where = append(where, fmt.Sprintf(
				"(tenant_id = $%d OR (is_core = true AND tenant_id = $%d))",
				len(args)-1, len(args)))
		} else {
			args = append(args, tenantID)
			where = append(where, fmt.Sprintf("tenant_id = $%d", len(args)))
		}
	}

	if boID != "" {
		args = append(args, boID)
		where = append(where, fmt.Sprintf("bo_id = $%d", len(args)))
	}
	if cur != nil {
		args = append(args, cur.UpdatedAt, cur.ID)
		where = append(where, fmt.Sprintf(
			"(updated_at, id::text) < ($%d::timestamptz, $%d)",
			len(args)-1, len(args)))
	}

	args = append(args, fetch)
	q := fmt.Sprintf(`
		SELECT %s
		FROM data_explorer.cube_definition
		WHERE %s
		ORDER BY updated_at DESC, id DESC
		LIMIT $%d
	`, cubeDefSelectCols, strings.Join(where, " AND "), len(args))

	var rows []cubeDefRow
	if err := h.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, nil, err
	}

	out := make([]CubeDefinition, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toCubeDefinition())
	}

	var next *cubeCursor
	if len(out) > limit {
		last := out[limit-1]
		out = out[:limit]
		next = &cubeCursor{UpdatedAt: last.UpdatedAt, ID: last.ID}
	}
	return out, next, nil
}

// HandleGetCube handles GET /api/cubes/{id}
func (h *CubeHandler) HandleGetCube(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, errors.New("cube id is required"), http.StatusBadRequest)
		return
	}
	c, err := h.getByID(r.Context(), secCtx.TenantID, id)
	if err == sql.ErrNoRows {
		h.writeError(w, errors.New("cube not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"cube": c})
}

// HandleCreateCube handles POST /api/cubes
func (h *CubeHandler) HandleCreateCube(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	raw, err := readLimitedBody(r, 1<<20)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	var req cubeWriteRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.BOID) == "" {
		h.writeError(w, errors.New("name and boId are required"), http.StatusBadRequest)
		return
	}

	c := req.applyCreate(secCtx.TenantID, secCtx.UserID)
	if c.IsCore && !h.isGoldCopy(r.Context(), secCtx.TenantID) {
		h.writeError(w, errors.New("only the gold-copy tenant may create core cubes"), http.StatusForbidden)
		return
	}
	if err := h.validateForWrite(r.Context(), secCtx.TenantID, &c); err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	out, err := h.insertCube(r.Context(), c)
	if err != nil {
		if isUniqueViolation(err) {
			h.writeError(w, fmt.Errorf("cube name %q already exists for this tenant", c.Name), http.StatusConflict)
			return
		}
		h.writeError(w, fmt.Errorf("create cube: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]interface{}{"cube": out})
}

// HandlePatchCube handles PATCH /api/cubes/{id}
//
// Non-breaking edits update in place. Breaking surface changes must use
// POST /api/cubes/{id}/versions so contract_version bumps explicitly.
// Unchanged content_hash is a no-op (same row returned).
func (h *CubeHandler) HandlePatchCube(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, errors.New("cube id is required"), http.StatusBadRequest)
		return
	}

	existing, err := h.getByID(r.Context(), secCtx.TenantID, id)
	if err == sql.ErrNoRows {
		h.writeError(w, errors.New("cube not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}
	if existing.TenantID != secCtx.TenantID {
		h.writeError(w, errors.New("cannot modify a core cube owned by the gold-copy tenant"), http.StatusForbidden)
		return
	}
	if existing.IsCore && !h.isGoldCopy(r.Context(), secCtx.TenantID) {
		h.writeError(w, errors.New("cannot modify a core cube"), http.StatusForbidden)
		return
	}

	raw, err := readLimitedBody(r, 1<<20)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	var req cubeWriteRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	draft := req.applyPatch(*existing)
	if bodyHasField(raw, "description") {
		draft.Description = req.Description
	}

	// Status-only patches skip structural/metric revalidation when surface unchanged.
	surfaceTouched := bodyHasField(raw, "dimensions") || bodyHasField(raw, "metricIds") ||
		bodyHasField(raw, "grains") || bodyHasField(raw, "federation") ||
		bodyHasField(raw, "timeDimension") || bodyHasField(raw, "materialization") ||
		bodyHasField(raw, "boId") || bodyHasField(raw, "name")

	if surfaceTouched {
		if reasons := DetectCubeContractBreaking(*existing, draft); len(reasons) > 0 {
			h.writeJSON(w, http.StatusConflict, map[string]interface{}{
				"error":          "breaking contract change",
				"details":        "use POST /api/cubes/{id}/versions to publish a new contract_version",
				"breakReasons":   reasons,
				"contractVersion": existing.ContractVersion,
			})
			return
		}
		if err := h.validateForWrite(r.Context(), secCtx.TenantID, &draft); err != nil {
			h.writeError(w, err, http.StatusBadRequest)
			return
		}
		if draft.ContentHash == existing.ContentHash {
			h.writeJSON(w, http.StatusOK, map[string]interface{}{
				"cube":  existing,
				"noop":  true,
				"reason": "content_hash unchanged",
			})
			return
		}
	} else if req.Status != "" {
		draft.Status = req.Status
		draft.ContentHash = existing.ContentHash
	} else {
		h.writeError(w, errors.New("no patch fields provided"), http.StatusBadRequest)
		return
	}

	out, err := h.updateCube(r.Context(), draft)
	if err != nil {
		if isUniqueViolation(err) {
			h.writeError(w, fmt.Errorf("cube name %q already exists for this tenant", draft.Name), http.StatusConflict)
			return
		}
		h.writeError(w, fmt.Errorf("patch cube: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"cube": out})
}

// HandleValidateCube handles POST /api/cubes/{id}/validate
//
// Runs structural + metric-reference checks without writing physicals.
// Optional body supplies a draft overlay; empty body validates the stored cube.
func (h *CubeHandler) HandleValidateCube(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, errors.New("cube id is required"), http.StatusBadRequest)
		return
	}

	existing, err := h.getByID(r.Context(), secCtx.TenantID, id)
	if err == sql.ErrNoRows {
		h.writeError(w, errors.New("cube not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	draft := *existing
	var keySamples []FederationKeySample
	raw, _ := readLimitedBody(r, 1<<20)
	if len(strings.TrimSpace(string(raw))) > 0 {
		var req cubeWriteRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
			return
		}
		draft = req.applyPatch(*existing)
		if bodyHasField(raw, "description") {
			draft.Description = req.Description
		}
		keySamples = req.FederationKeySamples
	}

	structuralOK := true
	var structuralErr string
	if err := ValidateCubeStructural(draft); err != nil {
		structuralOK = false
		structuralErr = err.Error()
	}

	metrics, err := h.loadMetrics(r.Context(), secCtx.TenantID, draft.MetricIDs)
	if err != nil {
		h.writeError(w, fmt.Errorf("load metrics: %w", err), http.StatusInternalServerError)
		return
	}
	metricsOK := true
	var metricsErr string
	if err := ValidateCubeMetricReferences(draft, metrics); err != nil {
		metricsOK = false
		metricsErr = err.Error()
	}

	metricIDSet := make(map[string]bool, len(metrics))
	for id := range metrics {
		metricIDSet[strings.ToLower(strings.TrimSpace(id))] = true
	}
	// Validate: require orphan samples only when the client supplied them.
	// Shape + transform compile always run for federated drafts.
	requireOrphan := len(keySamples) > 0 && !draft.Federation.Empty()
	projections, orphanReport, fedErr := EvaluateFederationPlan(
		draft.Federation, metricIDSet, keySamples, requireOrphan,
	)
	federationOK := fedErr == nil && (draft.Federation.Empty() || orphanReport.Ok || orphanReport.Skipped)
	var federationErr string
	if fedErr != nil {
		federationOK = false
		federationErr = fedErr.Error()
	} else if !draft.Federation.Empty() && !orphanReport.Skipped && !orphanReport.Ok {
		federationOK = false
		federationErr = orphanReport.Detail
	}

	breakReasons := DetectCubeContractBreaking(*existing, draft)
	hash := ComputeCubeContentHash(draft, metricContentHashes(metrics, draft.MetricIDs))

	ok := structuralOK && metricsOK && federationOK
	status := http.StatusOK
	if !ok {
		status = http.StatusUnprocessableEntity
	}
	h.writeJSON(w, status, map[string]interface{}{
		"ok":                   ok,
		"structuralOk":         structuralOK,
		"structuralError":      structuralErr,
		"metricsOk":            metricsOK,
		"metricsError":         metricsErr,
		"federationOk":         federationOK,
		"federationError":      federationErr,
		"federationTransforms": projections,
		"orphanReport":         orphanReport,
		"breakReasons":         breakReasons,
		"contentHash":          hash,
		"contractVersion":      existing.ContractVersion,
		"orphanRateMaxPct":     EffectiveOrphanRateMax(draft.Federation),
	})
}

// HandlePublishCubeVersion handles POST /api/cubes/{id}/versions
//
// Publishes a breaking contract change by bumping contract_version. Non-breaking
// drafts are rejected so callers use PATCH instead.
func (h *CubeHandler) HandlePublishCubeVersion(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, errors.New("cube id is required"), http.StatusBadRequest)
		return
	}

	existing, err := h.getByID(r.Context(), secCtx.TenantID, id)
	if err == sql.ErrNoRows {
		h.writeError(w, errors.New("cube not found"), http.StatusNotFound)
		return
	}
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}
	if existing.TenantID != secCtx.TenantID {
		h.writeError(w, errors.New("cannot version a core cube owned by the gold-copy tenant"), http.StatusForbidden)
		return
	}
	if existing.IsCore && !h.isGoldCopy(r.Context(), secCtx.TenantID) {
		h.writeError(w, errors.New("cannot version a core cube"), http.StatusForbidden)
		return
	}

	raw, err := readLimitedBody(r, 1<<20)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	var req cubeWriteRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	draft := req.applyPatch(*existing)
	if bodyHasField(raw, "description") {
		draft.Description = req.Description
	}
	reasons := DetectCubeContractBreaking(*existing, draft)
	if len(reasons) == 0 {
		h.writeError(w, errors.New("draft is not a breaking change; use PATCH /api/cubes/{id}"), http.StatusBadRequest)
		return
	}
	if err := h.validateForWrite(r.Context(), secCtx.TenantID, &draft); err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	draft.ContractVersion = existing.ContractVersion + 1

	out, err := h.updateCube(r.Context(), draft)
	if err != nil {
		h.writeError(w, fmt.Errorf("publish cube version: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"cube":         out,
		"breakReasons": reasons,
		"previousVersion": existing.ContractVersion,
	})
}

// cubeMetricOption is the governed-metric picker row for Cube Designer.
type cubeMetricOption struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	BOID         string `json:"boId"`
	ContentHash  string `json:"contentHash,omitempty"`
	Decomposable bool   `json:"decomposable"`
	IsCore       bool   `json:"isCore"`
	Status       string `json:"status"`
}

// HandleListCubeMetrics handles GET /api/cubes/metrics?boId=
//
// Returns active governed metrics (own-tenant + core) for the cube metric
// picker. Optional boId filters to metrics declared on that business object.
func (h *CubeHandler) HandleListCubeMetrics(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	boID := strings.TrimSpace(r.URL.Query().Get("boId"))
	args := []interface{}{secCtx.TenantID}
	where := `(tenant_id = $1 OR is_core = true) AND status = 'active' AND archived_at IS NULL`
	if boID != "" {
		args = append(args, boID)
		where += fmt.Sprintf(` AND bo_id = $%d`, len(args))
	}

	q := `
		SELECT id::text, name, COALESCE(description,'') AS description, bo_id,
		       COALESCE(content_hash,'') AS content_hash, decomposable, is_core, status
		FROM data_explorer.metric_definition
		WHERE ` + where + `
		ORDER BY name ASC
		LIMIT 500`

	type row struct {
		ID           string `db:"id"`
		Name         string `db:"name"`
		Description  string `db:"description"`
		BOID         string `db:"bo_id"`
		ContentHash  string `db:"content_hash"`
		Decomposable bool   `db:"decomposable"`
		IsCore       bool   `db:"is_core"`
		Status       string `db:"status"`
	}
	var rows []row
	if err := h.db.SelectContext(r.Context(), &rows, q, args...); err != nil {
		h.writeError(w, fmt.Errorf("list metrics: %w", err), http.StatusInternalServerError)
		return
	}
	out := make([]cubeMetricOption, 0, len(rows))
	for _, rw := range rows {
		out = append(out, cubeMetricOption{
			ID: rw.ID, Name: rw.Name, Description: rw.Description, BOID: rw.BOID,
			ContentHash: rw.ContentHash, Decomposable: rw.Decomposable,
			IsCore: rw.IsCore, Status: rw.Status,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"metrics": out})
}

func readLimitedBody(r *http.Request, max int64) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	limited := http.MaxBytesReader(nil, r.Body, max)
	b, err := io.ReadAll(limited)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, fmt.Errorf("request body too large")
		}
		return nil, err
	}
	return b, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "cube_definition_tenant_name_uniq")
}
