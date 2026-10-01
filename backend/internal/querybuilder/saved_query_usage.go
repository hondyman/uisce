package querybuilder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// SavedQueryReference describes one location where a saved query is used.
type SavedQueryReference struct {
	Type          string `json:"type"` // "page_draft" | "page_published" | "drill_through_target" | "core_adoption"
	ID            string `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	Location      string `json:"location,omitempty"`
	Version       int    `json:"version,omitempty"`
	SourceQueryID string `json:"sourceQueryId,omitempty"`
	Target        string `json:"target,omitempty"`
	TenantID      string `json:"tenantId,omitempty"`
	Count         int    `json:"count,omitempty"`
}

// SavedQueryUsageReport is the response payload for GET /api/explorer/saved-queries/{id}/usage
// and the 409 Conflict payload when deletion is rejected.
type SavedQueryUsageReport struct {
	InUse      bool                  `json:"inUse"`
	References []SavedQueryReference `json:"references"`
}

// FindSavedQueryReferencesInPageModel traverses an app_model (or generic page JSON)
// via a full recursive JSON walk to discover all queries, nested widget configs,
// containers, and drill-through targets referencing queryID at arbitrary depth.
func FindSavedQueryReferencesInPageModel(pageJSON []byte, queryID string, pageID, pageName string, isPublished bool, version int) []SavedQueryReference {
	if len(pageJSON) == 0 || queryID == "" {
		return nil
	}

	refType := "page_draft"
	if isPublished {
		refType = "page_published"
	}

	var root interface{}
	if err := json.Unmarshal(pageJSON, &root); err != nil {
		// Fallback simple string check if JSON parse fails
		if strings.Contains(string(pageJSON), queryID) {
			return []SavedQueryReference{{
				Type:     refType,
				ID:       pageID,
				Name:     pageName,
				Location: "page_model.raw",
				Version:  version,
			}}
		}
		return nil
	}

	var refs []SavedQueryReference
	findQueryReferencesRecursive(root, queryID, "", &refs, pageID, pageName, refType, version)
	return refs
}

func findQueryReferencesRecursive(v interface{}, queryID, path string, refs *[]SavedQueryReference, pageID, pageName string, refType string, version int) {
	if v == nil {
		return
	}
	switch val := v.(type) {
	case map[string]interface{}:
		// 1. Check direct query reference keys in this object
		sqID, _ := val["savedQueryId"].(string)
		qID, _ := val["queryId"].(string)
		if sqID == queryID || qID == queryID {
			*refs = append(*refs, SavedQueryReference{
				Type:     refType,
				ID:       pageID,
				Name:     pageName,
				Location: path,
				Version:  version,
			})
		}

		// 2. Check drill-through targets
		if target, ok := val["target"].(string); ok {
			if target == queryID || strings.HasSuffix(target, "/"+queryID) {
				*refs = append(*refs, SavedQueryReference{
					Type:     "drill_through_target",
					ID:       pageID,
					Name:     pageName,
					Location: path + ".target",
					Target:   target,
					Version:  version,
				})
			}
		}

		// 3. Recurse into all keys
		for k, child := range val {
			childPath := k
			if path != "" {
				childPath = path + "." + k
			}
			findQueryReferencesRecursive(child, queryID, childPath, refs, pageID, pageName, refType, version)
		}

	case []interface{}:
		// Recurse into all array elements
		for i, child := range val {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			findQueryReferencesRecursive(child, queryID, childPath, refs, pageID, pageName, refType, version)
		}
	}
}

type pageDefUsageRow struct {
	ID          string          `db:"id"`
	TenantID    string          `db:"tenant_id"`
	Name        string          `db:"name"`
	Slug        string          `db:"slug"`
	Status      string          `db:"status"`
	Version     int             `db:"version"`
	AppModel    json.RawMessage `db:"app_model"`
	Layout      json.RawMessage `db:"layout"`
	Components  json.RawMessage `db:"components"`
	DataSources json.RawMessage `db:"data_sources"`
}

type queryStateUsageRow struct {
	ID         string          `db:"id"`
	Name       string          `db:"name"`
	QueryState json.RawMessage `db:"query_state"`
}

type coreAdoptionUsageRow struct {
	TenantID string `db:"tenant_id"`
	Count    int    `db:"count"`
}

// ScanSavedQueryUsage performs a comprehensive server-side usage scan across:
// 1. Page definitions (draft and published)
// 2. Other saved queries (drill-through targets)
// 3. Core object adoptions (if core query)
func ScanSavedQueryUsage(ctx context.Context, db *sqlx.DB, queryID string, isCore bool, tenantID string) (SavedQueryUsageReport, error) {
	var refs []SavedQueryReference

	// 1. Scan page_definitions
	pageQuery := `SELECT id, tenant_id, name, slug, status, version,
	                     COALESCE(app_model, '{}'::jsonb) as app_model,
	                     COALESCE(layout, '[]'::jsonb) as layout,
	                     COALESCE(components, '[]'::jsonb) as components,
	                     COALESCE(data_sources, '[]'::jsonb) as data_sources
	              FROM page_definitions`
	var pageArgs []interface{}
	if !isCore && tenantID != "" {
		pageQuery += ` WHERE tenant_id = $1`
		pageArgs = append(pageArgs, tenantID)
	}

	var pages []pageDefUsageRow
	if err := db.SelectContext(ctx, &pages, pageQuery, pageArgs...); err == nil {
		for _, p := range pages {
			isPub := p.Status == "published"
			// Check app_model
			pRefs := FindSavedQueryReferencesInPageModel(p.AppModel, queryID, p.ID, p.Name, isPub, p.Version)
			refs = append(refs, pRefs...)

			// If not found in app_model, check components / data_sources
			if len(pRefs) == 0 {
				rawSources := string(p.DataSources)
				rawComps := string(p.Components)
				if strings.Contains(rawSources, queryID) || strings.Contains(rawComps, queryID) {
					refType := "page_draft"
					if isPub {
						refType = "page_published"
					}
					refs = append(refs, SavedQueryReference{
						Type:     refType,
						ID:       p.ID,
						Name:     p.Name,
						Location: "data_sources_or_components",
						Version:  p.Version,
					})
				}
			}
		}
	}

	// 2. Scan other saved queries for drill-through targets
	sqQuery := `SELECT id, name, query_state
	            FROM data_explorer.saved_query
	            WHERE id != $1 AND archived_at IS NULL`
	sqArgs := []interface{}{queryID}
	if !isCore && tenantID != "" {
		sqQuery += ` AND tenant_id = $2`
		sqArgs = append(sqArgs, tenantID)
	}

	var sqRows []queryStateUsageRow
	if err := db.SelectContext(ctx, &sqRows, sqQuery, sqArgs...); err == nil {
		for _, sq := range sqRows {
			rawState := string(sq.QueryState)
			if strings.Contains(rawState, queryID) {
				refs = append(refs, SavedQueryReference{
					Type:          "drill_through_target",
					SourceQueryID: sq.ID,
					Name:          sq.Name,
					Target:        queryID,
				})
			}
		}
	}

	// 3. If isCore, scan core_object_adoption
	if isCore {
		var adoptions []coreAdoptionUsageRow
		adoptQuery := `SELECT tenant_id, COUNT(*) as count
		               FROM public.core_object_adoption
		               WHERE object_type = 'saved_query' AND core_object_id = $1
		               GROUP BY tenant_id`
		if err := db.SelectContext(ctx, &adoptions, adoptQuery, queryID); err == nil {
			for _, a := range adoptions {
				refs = append(refs, SavedQueryReference{
					Type:     "core_adoption",
					TenantID: a.TenantID,
					Count:    a.Count,
				})
			}
		}
	}

	return SavedQueryUsageReport{
		InUse:      len(refs) > 0,
		References: refs,
	}, nil
}

// HandleGetSavedQueryUsage handles GET /api/explorer/saved-queries/{id}/usage.
func (h *SavedQueryHandler) HandleGetSavedQueryUsage(w http.ResponseWriter, r *http.Request) {
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}

	report, err := ScanSavedQueryUsage(r.Context(), h.db, sq.ID, sq.IsCore, tenantID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to scan saved query usage: %w", err), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, report)
}

// PatchSavedQueryRequest is the body for PATCH /api/explorer/saved-queries/{id}.
type PatchSavedQueryRequest struct {
	Status *string `json:"status,omitempty"` // "active" | "deprecated" | "archived"
}

// HandlePatchSavedQuery handles PATCH /api/explorer/saved-queries/{id}.
// Allows updating lifecycle status (e.g. deprecate a core or custom query).
func (h *SavedQueryHandler) HandlePatchSavedQuery(w http.ResponseWriter, r *http.Request) {
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}

	var req PatchSavedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	if req.Status == nil {
		h.writeError(w, fmt.Errorf("no patch fields specified"), http.StatusBadRequest)
		return
	}

	newStatus := *req.Status
	if newStatus != "active" && newStatus != "deprecated" && newStatus != "archived" {
		h.writeError(w, fmt.Errorf("invalid status %q: must be active, deprecated, or archived", newStatus), http.StatusBadRequest)
		return
	}

	// Core query protection: only gold copy tenant can deprecate/archive core query
	if sq.IsCore && !h.isGoldCopy(r.Context(), tenantID) {
		h.writeError(w, fmt.Errorf("cannot modify status of core query"), http.StatusForbidden)
		return
	}

	var archivedAt sql.NullTime
	if newStatus == "archived" {
		archivedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}

	var out savedQueryRow
	err := h.db.QueryRowxContext(r.Context(), `
		UPDATE data_explorer.saved_query
		SET status = $1, archived_at = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
		RETURNING `+savedQuerySelectCols+`
	`, newStatus, archivedAt, sq.ID, tenantID).StructScan(&out)

	if err != nil {
		h.writeError(w, fmt.Errorf("failed to update saved query status: %w", err), http.StatusInternalServerError)
		return
	}

	res := out.toSavedQuery()
	h.writeJSON(w, http.StatusOK, res)
}
