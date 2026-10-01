package querybuilder

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/corecustom"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/lib/pq"
)

const coreQueryObjectType = "saved_query"

// QueryCustomization represents how a client tenant adopts a core query.
type QueryCustomization struct {
	Mode             string  `json:"mode"` // vanilla | extended | cloned
	Active           bool    `json:"active"`
	CoreVersion      int     `json:"coreVersion"`
	BaseVersion      int     `json:"baseVersion,omitempty"`
	UpgradeAvailable bool    `json:"upgradeAvailable"`
	CloneQueryID     *string `json:"cloneQueryId,omitempty"`
}

// QueryCloneSource marks a tenant query as a clone of a core query.
type QueryCloneSource struct {
	QueryID string `json:"queryId"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type queryAdoption struct {
	CoreObjectID  string        `db:"core_object_id"`
	Active        bool          `db:"active"`
	Mode          string        `db:"mode"`
	BaseVersion   sql.NullInt64 `db:"base_version"`
	BaseSnapshot  []byte        `db:"base_snapshot"`
	Extension     []byte        `db:"extension"`
	CloneObjectID sql.NullString `db:"clone_object_id"`
}

const queryAdoptionCols = `core_object_id, active, mode, base_version, base_snapshot, extension, clone_object_id`

func (h *SavedQueryHandler) goldCopyID(ctx context.Context) string {
	tid := goldcopy.ResolveTenantID(ctx, h.db)
	if tid == uuid.Nil {
		return ""
	}
	return tid.String()
}

func (h *SavedQueryHandler) isGoldCopy(ctx context.Context, tenantID string) bool {
	gold := h.goldCopyID(ctx)
	return gold != "" && gold == tenantID
}

func (h *SavedQueryHandler) canCustomize(r *http.Request, tenantID string) bool {
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		return false
	}
	allowed := auth.IsGlobalAdmin
	for _, role := range auth.Roles {
		if role == "tenant_admin" || role == "admin" {
			allowed = true
		}
	}
	if !allowed {
		return false
	}
	gold := h.goldCopyID(r.Context())
	return gold != "" && gold != tenantID
}

func (h *SavedQueryHandler) adoptions(ctx context.Context, tenantID string) (map[string]queryAdoption, error) {
	var rows []queryAdoption
	query := `SELECT ` + queryAdoptionCols + ` FROM core_object_adoption WHERE tenant_id = $1 AND object_type = $2`
	if err := h.db.SelectContext(ctx, &rows, query, tenantID, coreQueryObjectType); err != nil {
		return nil, err
	}
	out := make(map[string]queryAdoption, len(rows))
	for _, a := range rows {
		out[a.CoreObjectID] = a
	}
	return out, nil
}

func (h *SavedQueryHandler) adoption(ctx context.Context, tenantID, coreID string) (*queryAdoption, error) {
	var a queryAdoption
	query := `SELECT ` + queryAdoptionCols + ` FROM core_object_adoption WHERE tenant_id = $1 AND object_type = $2 AND core_object_id = $3`
	err := h.db.GetContext(ctx, &a, query, tenantID, coreQueryObjectType, coreID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// computeContentHash returns a deterministic SHA-256 hash of the normalized query content.
func computeContentHash(c savedQueryContent) string {
	b, _ := json.Marshal(c.normalized())
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// presentCoreQuery transforms a gold-copy core query into what the calling tenant sees.
func presentCoreQuery(sq *SavedQuery, a *queryAdoption, isMaster bool, canCustomize bool) {
	if isMaster {
		sq.CoreStatus = "core"
		sq.Editable = true
		sq.CanCustomize = false
		return
	}

	sq.Editable = false
	sq.CanCustomize = canCustomize

	// Read-time resolution for vanilla when no adoption row exists
	if a == nil || a.Mode == "vanilla" {
		sq.CoreStatus = "vanilla"
		sq.Customization = &QueryCustomization{
			Mode:        "vanilla",
			Active:      a == nil || a.Active,
			CoreVersion: 1,
		}
		return
	}

	c := &QueryCustomization{
		Mode:   a.Mode,
		Active: a.Active,
	}
	if a.BaseVersion.Valid {
		c.BaseVersion = int(a.BaseVersion.Int64)
	}

	switch a.Mode {
	case "extended":
		// Compute current core hash before applying extension
		curCoreHash := computeContentHash(contentFromSavedQuery(sq))
		baseHash := ""
		if len(a.BaseSnapshot) > 0 {
			if baseC, err := contentFromJSON(a.BaseSnapshot); err == nil {
				baseHash = computeContentHash(baseC)
			}
		}
		c.UpgradeAvailable = (baseHash != "" && baseHash != curCoreHash)

		if ext, err := contentFromJSON(a.Extension); err == nil {
			sq.Name = ext.Name
			sq.Description = ext.Description
			sq.RelatedBOIDs = ext.RelatedBOIDs
			sq.ChartType = ext.ChartType
			sq.State = ext.State
			sq.Tags = ext.Tags
		}
		if c.UpgradeAvailable {
			sq.CoreStatus = "upgrade_available"
		} else {
			sq.CoreStatus = "extended"
		}
	case "cloned":
		sq.CoreStatus = "cloned"
		if a.CloneObjectID.Valid {
			id := a.CloneObjectID.String
			c.CloneQueryID = &id
		}
	default:
		sq.CoreStatus = "vanilla"
	}

	sq.Customization = c
}

// validateAdoptionPreflight checks that the core query's BO, related BOs, and referenced semantic termNodeIds resolve in the tenant.
func (h *SavedQueryHandler) validateAdoptionPreflight(tenantID string, content savedQueryContent) error {
	var unresolvable []string
	if owned, err := h.service.BOBelongsToTenant(content.BOID, tenantID); err != nil || !owned {
		unresolvable = append(unresolvable, fmt.Sprintf("primary BO %q", content.BOID))
	}
	for _, relBO := range content.RelatedBOIDs {
		if owned, err := h.service.BOBelongsToTenant(relBO, tenantID); err != nil || !owned {
			unresolvable = append(unresolvable, fmt.Sprintf("related BO %q", relBO))
		}
	}

	// Verify all termNodeIds in dimensions, measures, and filters
	termSet := make(map[string]bool)
	for _, d := range content.State.Dimensions {
		if d.TermNodeID != "" {
			termSet[d.TermNodeID] = true
		}
	}
	for _, m := range content.State.Measures {
		if m.TermNodeID != "" {
			termSet[m.TermNodeID] = true
		}
	}
	for _, f := range content.State.Filters {
		if f.TermNodeID != "" {
			termSet[f.TermNodeID] = true
		}
	}

	// If there are terms, verify they exist in catalog_node or term registry for tenant
	if len(termSet) > 0 {
		var count int
		var termIDs []string
		for tid := range termSet {
			termIDs = append(termIDs, tid)
		}
		// Query catalog_node table for term existence if catalog table is present
		query := `SELECT COUNT(DISTINCT id) FROM catalog_node WHERE id = ANY($1)`
		if err := h.db.Get(&count, query, pq.Array(termIDs)); err == nil {
			if count < len(termIDs) {
				// Query which specific terms are missing
				var foundIDs []string
				_ = h.db.Select(&foundIDs, `SELECT id FROM catalog_node WHERE id = ANY($1)`, pq.Array(termIDs))
				foundMap := make(map[string]bool, len(foundIDs))
				for _, fid := range foundIDs {
					foundMap[fid] = true
				}
				for _, tid := range termIDs {
					if !foundMap[tid] {
						unresolvable = append(unresolvable, fmt.Sprintf("semantic term %q", tid))
					}
				}
			}
		}
	}

	if len(unresolvable) > 0 {
		return fmt.Errorf("cannot adopt core query: the following objects do not resolve in your tenant environment: %v", unresolvable)
	}
	return nil
}

// HandleExtendCoreQuery handles POST /api/explorer/saved-queries/{id}/extend.
func (h *SavedQueryHandler) HandleExtendCoreQuery(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	if !h.canCustomize(r, secCtx.TenantID) {
		h.writeError(w, errors.New("customizing core queries requires tenant admin role outside master tenant"), http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	gold := h.goldCopyID(r.Context())
	if gold == "" {
		h.writeError(w, errors.New("no gold copy master tenant configured"), http.StatusInternalServerError)
		return
	}

	// Load the core query from the gold copy tenant
	var coreRow savedQueryRow
	err = h.db.Get(&coreRow, `SELECT `+savedQuerySelectCols+` FROM data_explorer.saved_query WHERE id = $1 AND is_core = true AND tenant_id = $2`, id, gold)
	if err != nil {
		h.writeError(w, fmt.Errorf("core query not found: %w", err), http.StatusNotFound)
		return
	}
	core := coreRow.toSavedQuery()
	baseContent := contentFromSavedQuery(&core)

	// Decode requested extension
	var req savedQueryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	extContent := savedQueryContent{
		Name:         req.Name,
		Description:  req.Description,
		BOID:         core.BOID,
		BindingID:    core.BindingID,
		RelatedBOIDs: req.RelatedBOIDs,
		ChartType:    req.ChartType,
		State:        req.State,
		Tags:         req.Tags,
	}.normalized()

	// 1. Preflight check: BO resolution in client tenant
	if err := h.validateAdoptionPreflight(secCtx.TenantID, extContent); err != nil {
		h.writeError(w, err, http.StatusUnprocessableEntity)
		return
	}

	// 2. Additive-Only validation check
	if err := ValidateAdditiveExtension(baseContent, extContent); err != nil {
		h.writeError(w, fmt.Errorf("additive contract violation: %w", err), http.StatusBadRequest)
		return
	}

	baseBytes, _ := json.Marshal(baseContent)
	extBytes, _ := json.Marshal(extContent)

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO core_object_adoption (tenant_id, object_type, core_object_id, mode, base_version, base_snapshot, extension, updated_at, updated_by)
		VALUES ($1, $2, $3, 'extended', 1, $4, $5, NOW(), $6)
		ON CONFLICT (tenant_id, object_type, core_object_id) DO UPDATE SET
			mode = 'extended',
			base_version = CASE WHEN core_object_adoption.mode = 'extended' THEN core_object_adoption.base_version ELSE EXCLUDED.base_version END,
			base_snapshot = CASE WHEN core_object_adoption.mode = 'extended' THEN core_object_adoption.base_snapshot ELSE EXCLUDED.base_snapshot END,
			extension = EXCLUDED.extension,
			updated_at = NOW(), updated_by = EXCLUDED.updated_by
	`, secCtx.TenantID, coreQueryObjectType, core.ID, baseBytes, extBytes, secCtx.UserID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to save query extension: %w", err), http.StatusInternalServerError)
		return
	}

	a, _ := h.adoption(r.Context(), secCtx.TenantID, core.ID)
	presentCoreQuery(&core, a, false, true)
	h.writeJSON(w, http.StatusOK, core)
}

// QueryComparison is the response format for comparing core updates against tenant extensions.
type QueryComparison struct {
	BaseVersion      int                `json:"baseVersion"`
	CoreVersion      int                `json:"coreVersion"`
	UpgradeAvailable bool               `json:"upgradeAvailable"`
	Customizations   []corecustom.Group `json:"customizations"`
	CoreUpdates      []corecustom.Group `json:"coreUpdates"`
	Conflicts        []string           `json:"conflicts,omitempty"`
}

func (h *SavedQueryHandler) loadCoreAndAdoption(w http.ResponseWriter, r *http.Request) (string, *SavedQuery, *queryAdoption, bool) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return "", nil, nil, false
	}
	id := chi.URLParam(r, "id")
	gold := h.goldCopyID(r.Context())
	if gold == "" {
		h.writeError(w, errors.New("no gold copy tenant configured"), http.StatusInternalServerError)
		return "", nil, nil, false
	}

	var coreRow savedQueryRow
	err = h.db.Get(&coreRow, `SELECT `+savedQuerySelectCols+` FROM data_explorer.saved_query WHERE id = $1 AND is_core = true AND tenant_id = $2`, id, gold)
	if err != nil {
		h.writeError(w, fmt.Errorf("core query not found: %w", err), http.StatusNotFound)
		return "", nil, nil, false
	}
	core := coreRow.toSavedQuery()

	a, err := h.adoption(r.Context(), secCtx.TenantID, core.ID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to load adoption: %w", err), http.StatusInternalServerError)
		return "", nil, nil, false
	}
	return secCtx.TenantID, &core, a, true
}

// HandleCompareCoreQuery handles GET /api/explorer/saved-queries/{id}/compare.
func (h *SavedQueryHandler) HandleCompareCoreQuery(w http.ResponseWriter, r *http.Request) {
	_, core, a, ok := h.loadCoreAndAdoption(w, r)
	if !ok {
		return
	}
	if a == nil || a.Mode != "extended" || len(a.BaseSnapshot) == 0 {
		h.writeJSON(w, http.StatusOK, QueryComparison{
			BaseVersion: 1, CoreVersion: 1, UpgradeAvailable: false,
			Customizations: []corecustom.Group{}, CoreUpdates: []corecustom.Group{},
		})
		return
	}

	baseContent, err := contentFromJSON(a.BaseSnapshot)
	if err != nil {
		h.writeError(w, fmt.Errorf("corrupt base snapshot: %w", err), http.StatusInternalServerError)
		return
	}
	extContent, err := contentFromJSON(a.Extension)
	if err != nil {
		h.writeError(w, fmt.Errorf("corrupt extension: %w", err), http.StatusInternalServerError)
		return
	}
	curContent := contentFromSavedQuery(core)

	rep := corecustom.Compare(baseContent.doc(), extContent.doc(), curContent.doc(), savedQueryGrouper)

	// Identify any alias collisions as conflicts
	var conflicts []string
	extAliases := make(map[string]bool)
	for _, d := range extContent.State.Dimensions {
		extAliases[d.Alias] = true
	}
	for _, m := range extContent.State.Measures {
		extAliases[m.Alias] = true
	}
	for _, d := range curContent.State.Dimensions {
		if extAliases[d.Alias] && !containsDimAlias(baseContent.State.Dimensions, d.Alias) {
			conflicts = append(conflicts, fmt.Sprintf("core update added dimension %q colliding with tenant extension", d.Alias))
		}
	}
	for _, m := range curContent.State.Measures {
		if extAliases[m.Alias] && !containsMeasAlias(baseContent.State.Measures, m.Alias) {
			conflicts = append(conflicts, fmt.Sprintf("core update added measure %q colliding with tenant extension", m.Alias))
		}
	}

	h.writeJSON(w, http.StatusOK, QueryComparison{
		BaseVersion:      1,
		CoreVersion:      1,
		UpgradeAvailable: len(rep.CoreUpdates) > 0,
		Customizations:   nonNilGroups(rep.Customizations),
		CoreUpdates:      nonNilGroups(rep.CoreUpdates),
		Conflicts:        conflicts,
	})
}

func containsDimAlias(dims []SavedQueryDimension, alias string) bool {
	for _, d := range dims {
		if d.Alias == alias {
			return true
		}
	}
	return false
}

func containsMeasAlias(measures []SavedQueryMeasure, alias string) bool {
	for _, m := range measures {
		if m.Alias == alias {
			return true
		}
	}
	return false
}

func nonNilGroups(g []corecustom.Group) []corecustom.Group {
	if g == nil {
		return []corecustom.Group{}
	}
	return g
}

// HandleUpgradeCoreQuery handles POST /api/explorer/saved-queries/{id}/upgrade?dryRun=true|false.
func (h *SavedQueryHandler) HandleUpgradeCoreQuery(w http.ResponseWriter, r *http.Request) {
	tenantID, core, a, ok := h.loadCoreAndAdoption(w, r)
	if !ok {
		return
	}
	if a == nil || a.Mode != "extended" || len(a.BaseSnapshot) == 0 {
		h.writeError(w, errors.New("no active extension to upgrade"), http.StatusConflict)
		return
	}

	dryRun := r.URL.Query().Get("dryRun") == "true"

	var req struct {
		Remove []string `json:"remove"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	removeMap := make(map[string]bool, len(req.Remove))
	for _, id := range req.Remove {
		removeMap[id] = true
	}

	baseContent, err := contentFromJSON(a.BaseSnapshot)
	if err != nil {
		h.writeError(w, fmt.Errorf("corrupt base snapshot: %w", err), http.StatusInternalServerError)
		return
	}
	extContent, err := contentFromJSON(a.Extension)
	if err != nil {
		h.writeError(w, fmt.Errorf("corrupt extension: %w", err), http.StatusInternalServerError)
		return
	}
	curContent := contentFromSavedQuery(core)

	mergedDoc := corecustom.Merge(baseContent.doc(), extContent.doc(), curContent.doc(), removeMap, savedQueryGrouper)
	mergedContent, err := contentFromDoc(mergedDoc)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to merge: %w", err), http.StatusInternalServerError)
		return
	}

	// Validate merged output against current core using Additive validator
	if err := ValidateAdditiveExtension(curContent, mergedContent); err != nil {
		h.writeError(w, fmt.Errorf("merged result violates additive contract: %w", err), http.StatusConflict)
		return
	}

	if dryRun {
		preview := *core
		preview.Name = mergedContent.Name
		preview.Description = mergedContent.Description
		preview.RelatedBOIDs = mergedContent.RelatedBOIDs
		preview.ChartType = mergedContent.ChartType
		preview.State = mergedContent.State
		preview.Tags = mergedContent.Tags
		h.writeJSON(w, http.StatusOK, map[string]interface{}{"dryRun": true, "preview": preview})
		return
	}

	newBaseBytes, _ := json.Marshal(curContent)
	newExtBytes, _ := json.Marshal(mergedContent)

	_, err = h.db.ExecContext(r.Context(), `
		UPDATE core_object_adoption
		SET base_version = 1, base_snapshot = $1, extension = $2, updated_at = NOW(), updated_by = $3
		WHERE tenant_id = $4 AND object_type = $5 AND core_object_id = $6
	`, newBaseBytes, newExtBytes, userOf(r), tenantID, coreQueryObjectType, core.ID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to save upgraded extension: %w", err), http.StatusInternalServerError)
		return
	}

	aFresh, _ := h.adoption(r.Context(), tenantID, core.ID)
	presentCoreQuery(core, aFresh, false, true)
	h.writeJSON(w, http.StatusOK, core)
}

// HandleRevertCoreQuery handles POST /api/explorer/saved-queries/{id}/revert.
// Sets mode to 'vanilla' while keeping extension dormant in DB for easy re-extension.
func (h *SavedQueryHandler) HandleRevertCoreQuery(w http.ResponseWriter, r *http.Request) {
	tenantID, core, a, ok := h.loadCoreAndAdoption(w, r)
	if !ok {
		return
	}
	if a != nil {
		_, err := h.db.ExecContext(r.Context(), `
			UPDATE core_object_adoption
			SET mode = 'vanilla', updated_at = NOW(), updated_by = $1
			WHERE tenant_id = $2 AND object_type = $3 AND core_object_id = $4
		`, userOf(r), tenantID, coreQueryObjectType, core.ID)
		if err != nil {
			h.writeError(w, fmt.Errorf("failed to revert: %w", err), http.StatusInternalServerError)
			return
		}
	}
	aFresh, _ := h.adoption(r.Context(), tenantID, core.ID)
	presentCoreQuery(core, aFresh, false, true)
	h.writeJSON(w, http.StatusOK, core)
}

func userOf(r *http.Request) string {
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok {
		return auth.UserID
	}
	return ""
}
