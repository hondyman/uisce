package querybuilder

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/lib/pq"
)

// QueryBundleSchemaVersion is the official bundle schema identifier.
const QueryBundleSchemaVersion = "uisce.query-bundle/1"

type BundleSourceEnvironment struct {
	TenantID string `json:"tenantId,omitempty"`
	Product  string `json:"product,omitempty"`
}

type BundleQueryProvenance struct {
	IsCore      bool   `json:"isCore"`
	OriginalID  string `json:"originalId"`
	ContentHash string `json:"contentHash"`
}

type BundleQueryDependencies struct {
	BOID         string   `json:"boId"`
	RelatedBOIDs []string `json:"relatedBoIds"`
	TermNodeIDs  []string `json:"termNodeIds"`
}

type BundleQueryItem struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description,omitempty"`
	BOID         string                  `json:"boId"`
	BindingID    string                  `json:"bindingId,omitempty"`
	RelatedBOIDs []string                `json:"relatedBoIds,omitempty"`
	ChartType    string                  `json:"chartType"`
	State        SavedQueryState         `json:"state"`
	Tags         []string                `json:"tags,omitempty"`
	Provenance   BundleQueryProvenance   `json:"provenance"`
	Dependencies BundleQueryDependencies `json:"dependencies"`
}

type QueryBundle struct {
	SchemaVersion     string                  `json:"schemaVersion"`
	ExportedAt        time.Time               `json:"exportedAt"`
	SourceEnvironment BundleSourceEnvironment `json:"sourceEnvironment,omitempty"`
	Queries           []BundleQueryItem       `json:"queries"`
	TermMappings      map[string]string       `json:"termMappings,omitempty"`
}

type ExportSavedQueriesBatchRequest struct {
	IDs []string `json:"ids"`
}

type ImportSavedQueryRequest struct {
	Bundle    QueryBundle `json:"bundle"`
	Overwrite bool        `json:"overwrite,omitempty"`
	IsCore    *bool       `json:"isCore,omitempty"`
}

type ImportItemResult struct {
	OriginalID  string `json:"originalId"`
	NewID       string `json:"newId,omitempty"`
	Name        string `json:"name"`
	Status      string `json:"status"` // "imported" | "skipped" | "renamed" | "overwritten"
	Reason      string `json:"reason,omitempty"`
	ContentHash string `json:"contentHash"`
}

type ImportSavedQueryReport struct {
	ImportedCount int                `json:"importedCount"`
	SkippedCount  int                `json:"skippedCount"`
	Items         []ImportItemResult `json:"items"`
}

// ComputeContentHash returns the deterministic SHA-256 hash of a query's canonical content.
func ComputeContentHash(c savedQueryContent) string {
	b, _ := json.Marshal(c.normalized())
	h := sha256.Sum256(b)
	return fmt.Sprintf("sha256:%x", h)
}

// extractDependencies gathers all termNodeIds and related BO IDs from a SavedQuery.
func extractDependencies(sq *SavedQuery) BundleQueryDependencies {
	termSet := make(map[string]bool)
	for _, d := range sq.State.Dimensions {
		if d.TermNodeID != "" {
			termSet[d.TermNodeID] = true
		}
	}
	for _, m := range sq.State.Measures {
		if m.TermNodeID != "" {
			termSet[m.TermNodeID] = true
		}
	}
	for _, f := range sq.State.Filters {
		if f.TermNodeID != "" {
			termSet[f.TermNodeID] = true
		}
	}

	terms := make([]string, 0, len(termSet))
	for t := range termSet {
		terms = append(terms, t)
	}

	relBOs := sq.RelatedBOIDs
	if relBOs == nil {
		relBOs = []string{}
	}

	return BundleQueryDependencies{
		BOID:         sq.BOID,
		RelatedBOIDs: relBOs,
		TermNodeIDs:  terms,
	}
}

func buildBundleQueryItem(sq *SavedQuery) BundleQueryItem {
	content := contentFromSavedQuery(sq)
	hash := ComputeContentHash(content)
	deps := extractDependencies(sq)

	tags := sq.Tags
	if tags == nil {
		tags = []string{}
	}
	relBOs := sq.RelatedBOIDs
	if relBOs == nil {
		relBOs = []string{}
	}

	return BundleQueryItem{
		ID:           sq.ID,
		Name:         sq.Name,
		Description:  sq.Description,
		BOID:         sq.BOID,
		BindingID:    sq.BindingID,
		RelatedBOIDs: relBOs,
		ChartType:    sq.ChartType,
		State:        sq.State,
		Tags:         tags,
		Provenance: BundleQueryProvenance{
			IsCore:      sq.IsCore,
			OriginalID:  sq.ID,
			ContentHash: hash,
		},
		Dependencies: deps,
	}
}

// HandleExportSavedQuery handles GET /api/explorer/saved-queries/{id}/export.
func (h *SavedQueryHandler) HandleExportSavedQuery(w http.ResponseWriter, r *http.Request) {
	sq, tenantID, ok := h.loadOwned(w, r)
	if !ok {
		return
	}

	item := buildBundleQueryItem(sq)
	termMappings := make(map[string]string)
	for _, t := range item.Dependencies.TermNodeIDs {
		termMappings[t] = fmt.Sprintf("%s.%s", sq.BOID, t)
	}

	bundle := QueryBundle{
		SchemaVersion: QueryBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		SourceEnvironment: BundleSourceEnvironment{
			TenantID: tenantID,
		},
		Queries:      []BundleQueryItem{item},
		TermMappings: termMappings,
	}

	h.writeJSON(w, http.StatusOK, bundle)
}

// HandleExportSavedQueriesBatch handles POST /api/explorer/saved-queries/export.
func (h *SavedQueryHandler) HandleExportSavedQueriesBatch(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	var req ExportSavedQueriesBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		h.writeError(w, errors.New("no query IDs specified"), http.StatusBadRequest)
		return
	}

	items := make([]BundleQueryItem, 0, len(req.IDs))
	termMappings := make(map[string]string)

	for _, id := range req.IDs {
		var row savedQueryRow
		err := h.db.Get(&row, `SELECT `+savedQuerySelectCols+`
		                      FROM data_explorer.saved_query
		                      WHERE id = $1 AND tenant_id = $2 AND (visibility = 'shared' OR user_id = $3)`,
			id, secCtx.TenantID, secCtx.UserID)
		if err != nil {
			// Check core query fallback
			gold := h.goldCopyID(r.Context())
			if gold != "" {
				var coreRow savedQueryRow
				errCore := h.db.Get(&coreRow, `SELECT `+savedQuerySelectCols+`
				                              FROM data_explorer.saved_query
				                              WHERE id = $1 AND is_core = true AND tenant_id = $2`,
					id, gold)
				if errCore != nil {
					h.writeError(w, fmt.Errorf("saved query %q not found or not accessible: %w", id, err), http.StatusNotFound)
					return
				}
				row = coreRow
			} else {
				h.writeError(w, fmt.Errorf("saved query %q not found or not accessible: %w", id, err), http.StatusNotFound)
				return
			}
		}

		sq := row.toSavedQuery()
		item := buildBundleQueryItem(&sq)
		items = append(items, item)
		for _, t := range item.Dependencies.TermNodeIDs {
			termMappings[t] = fmt.Sprintf("%s.%s", sq.BOID, t)
		}
	}

	bundle := QueryBundle{
		SchemaVersion: QueryBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		SourceEnvironment: BundleSourceEnvironment{
			TenantID: secCtx.TenantID,
		},
		Queries:      items,
		TermMappings: termMappings,
	}

	h.writeJSON(w, http.StatusOK, bundle)
}

// HandleImportSavedQueries handles POST /api/explorer/saved-queries/import.
func (h *SavedQueryHandler) HandleImportSavedQueries(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	var req ImportSavedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}

	if req.Bundle.SchemaVersion != QueryBundleSchemaVersion {
		h.writeError(w, fmt.Errorf("unsupported bundle schemaVersion %q; expected %q", req.Bundle.SchemaVersion, QueryBundleSchemaVersion), http.StatusBadRequest)
		return
	}

	isMaster := h.isGoldCopy(r.Context(), secCtx.TenantID)
	// Core query import check: client tenant attempting to set is_core = true -> 403 Forbidden
	importAsCore := false
	if req.IsCore != nil && *req.IsCore {
		if !isMaster {
			h.writeError(w, errors.New("only master gold copy tenant can import queries as core"), http.StatusForbidden)
			return
		}
		importAsCore = true
	} else if isMaster && len(req.Bundle.Queries) > 0 && req.Bundle.Queries[0].Provenance.IsCore {
		importAsCore = true
	}

	// -------------------------------------------------------------
	// Phase A: All-or-Nothing Preflight Validation
	// -------------------------------------------------------------
	for _, q := range req.Bundle.Queries {
		if strings.TrimSpace(q.Name) == "" {
			h.writeError(w, fmt.Errorf("query with id %q is missing a name", q.ID), http.StatusUnprocessableEntity)
			return
		}
		if strings.TrimSpace(q.BOID) == "" {
			h.writeError(w, fmt.Errorf("query %q is missing required primary boId", q.Name), http.StatusUnprocessableEntity)
			return
		}
	}

	// -------------------------------------------------------------
	// Phase B: Load Existing Tenant Queries for Dedupe & Collision
	// -------------------------------------------------------------
	var existingRows []savedQueryRow
	_ = h.db.SelectContext(r.Context(), &existingRows, `
		SELECT `+savedQuerySelectCols+`
		FROM data_explorer.saved_query
		WHERE tenant_id = $1 AND archived_at IS NULL
	`, secCtx.TenantID)

	existingByHash := make(map[string]savedQueryRow)
	existingByName := make(map[string]savedQueryRow)

	for _, row := range existingRows {
		sq := row.toSavedQuery()
		hash := ComputeContentHash(contentFromSavedQuery(&sq))
		existingByHash[hash] = row
		existingByName[row.Name] = row
	}

	report := ImportSavedQueryReport{
		Items: make([]ImportItemResult, 0, len(req.Bundle.Queries)),
	}

	tx, err := h.db.Beginx()
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to begin transaction: %w", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	for _, item := range req.Bundle.Queries {
		// Apply termMappings if provided by the bundle
		if len(req.Bundle.TermMappings) > 0 {
			for i, d := range item.State.Dimensions {
				if mapped, ok := req.Bundle.TermMappings[d.TermNodeID]; ok && mapped != "" {
					item.State.Dimensions[i].TermNodeID = mapped
				}
			}
			for i, m := range item.State.Measures {
				if mapped, ok := req.Bundle.TermMappings[m.TermNodeID]; ok && mapped != "" {
					item.State.Measures[i].TermNodeID = mapped
				}
			}
			for i, f := range item.State.Filters {
				if mapped, ok := req.Bundle.TermMappings[f.TermNodeID]; ok && mapped != "" {
					item.State.Filters[i].TermNodeID = mapped
				}
			}
		}

		content := savedQueryContent{
			Name:         item.Name,
			Description:  item.Description,
			BOID:         item.BOID,
			BindingID:    item.BindingID,
			RelatedBOIDs: item.RelatedBOIDs,
			ChartType:    item.ChartType,
			State:        item.State,
			Tags:         item.Tags,
		}.normalized()

		itemHash := ComputeContentHash(content)

		// 1. Dedupe check: exact content hash already present in tenant
		if existing, exists := existingByHash[itemHash]; exists {
			report.SkippedCount++
			report.Items = append(report.Items, ImportItemResult{
				OriginalID:  item.ID,
				NewID:       existing.ID,
				Name:        existing.Name,
				Status:      "skipped",
				Reason:      "already-present",
				ContentHash: itemHash,
			})
			continue
		}

		// 2. Name collision check: same name exists with different content hash
		var finalName = item.Name
		var itemStatus = "imported"
		var targetID = uuid.NewString()

		if existing, nameExists := existingByName[item.Name]; nameExists {
			if req.Overwrite {
				// Reject overwrite on core query or core adoption
				if existing.IsCore {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error":   "Conflict",
						"message": fmt.Sprintf("cannot overwrite adopted core query %q", existing.Name),
					})
					return
				}

				// Usage scan before overwrite to protect pages
				usage, err := ScanSavedQueryUsage(r.Context(), h.db, existing.ID, existing.IsCore, secCtx.TenantID)
				if err == nil && usage.InUse {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error":      "Conflict",
						"message":    fmt.Sprintf("cannot overwrite query %q because it is in use by %d reference(s)", existing.Name, len(usage.References)),
						"references": usage.References,
					})
					return
				}

				targetID = existing.ID
				itemStatus = "overwritten"
				stateBytes, _ := json.Marshal(item.State)
				_, err = tx.ExecContext(r.Context(), `
					UPDATE data_explorer.saved_query
					SET description = $1, binding_id = $2, related_bo_ids = $3, chart_type = $4,
					    query_state = $5, tags = $6, updated_at = NOW()
					WHERE id = $7 AND tenant_id = $8
				`, item.Description, item.BindingID, pq.Array(item.RelatedBOIDs), item.ChartType,
					stateBytes, pq.Array(item.Tags), existing.ID, secCtx.TenantID)
				if err != nil {
					h.writeError(w, fmt.Errorf("failed to overwrite saved query %q: %w", existing.Name, err), http.StatusInternalServerError)
					return
				}
			} else {
				// Rename with (imported) suffix
				finalName = fmt.Sprintf("%s (imported)", item.Name)
				count := 1
				for {
					if _, taken := existingByName[finalName]; !taken {
						break
					}
					count++
					finalName = fmt.Sprintf("%s (imported %d)", item.Name, count)
				}
				itemStatus = "renamed"
			}
		}

		if itemStatus != "overwritten" {
			stateBytes, _ := json.Marshal(item.State)
			relBOs := item.RelatedBOIDs
			if relBOs == nil {
				relBOs = []string{}
			}
			tags := item.Tags
			if tags == nil {
				tags = []string{}
			}

			_, err = tx.ExecContext(r.Context(), `
				INSERT INTO data_explorer.saved_query
					(id, tenant_id, user_id, name, description, source_kind, source_id, binding_id, related_bo_ids, chart_type, query_state, tags, is_core, created_by)
				VALUES ($1, $2, $3, $4, $5, 'business_object', $6, $7, $8, $9, $10, $11, $12, $3)
			`, targetID, secCtx.TenantID, secCtx.UserID, finalName, item.Description,
				item.BOID, item.BindingID, pq.Array(relBOs), item.ChartType,
				stateBytes, pq.Array(tags), importAsCore)

			if err != nil {
				h.writeError(w, fmt.Errorf("failed to insert imported query %q: %w", finalName, err), http.StatusInternalServerError)
				return
			}
		}

		existingByName[finalName] = savedQueryRow{ID: targetID, Name: finalName}
		existingByHash[itemHash] = savedQueryRow{ID: targetID, Name: finalName}

		report.ImportedCount++
		report.Items = append(report.Items, ImportItemResult{
			OriginalID:  item.ID,
			NewID:       targetID,
			Name:        finalName,
			Status:      itemStatus,
			ContentHash: itemHash,
		})
	}

	if err := tx.Commit(); err != nil {
		h.writeError(w, fmt.Errorf("failed to commit import transaction: %w", err), http.StatusInternalServerError)
		return
	}
	tx = nil

	h.writeJSON(w, http.StatusOK, report)
}
