package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PageBundleSchemaVersion is the official page bundle schema identifier.
const PageBundleSchemaVersion = "uisce.page-bundle/1"

type PageSourceEnvironment struct {
	TenantID string `json:"tenantId,omitempty"`
	Product  string `json:"product,omitempty"`
}

type PageBundleDefinition struct {
	Name               string           `json:"name"`
	Slug               string           `json:"slug"`
	Description        string           `json:"description,omitempty"`
	Layout             json.RawMessage  `json:"layout"`
	Tabs               json.RawMessage  `json:"tabs,omitempty"`
	Components         json.RawMessage  `json:"components"`
	DataSources        json.RawMessage  `json:"dataSources"`
	PresentationEvents json.RawMessage  `json:"presentationEvents,omitempty"`
	FilterBar          json.RawMessage  `json:"filterBar,omitempty"`
	App                *json.RawMessage `json:"app,omitempty"`
}

type PageBundleProvenance struct {
	IsCore      bool   `json:"isCore"`
	OriginalID  string `json:"originalId"`
	ContentHash string `json:"contentHash"`
}

type PageBundleQueryDependencies struct {
	BOID         string   `json:"boId"`
	RelatedBOIDs []string `json:"relatedBoIds"`
	TermNodeIDs  []string `json:"termNodeIds"`
}

type PageBundleQueryItem struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	Description  string                      `json:"description,omitempty"`
	BOID         string                      `json:"boId"`
	BindingID    string                      `json:"bindingId,omitempty"`
	RelatedBOIDs []string                    `json:"relatedBoIds,omitempty"`
	ChartType    string                      `json:"chartType"`
	State        json.RawMessage             `json:"state"`
	Tags         []string                    `json:"tags,omitempty"`
	Provenance   PageBundleProvenance        `json:"provenance"`
	Dependencies PageBundleQueryDependencies `json:"dependencies"`
}

type PageBundleDependencies struct {
	Queries      []PageBundleQueryItem `json:"queries"`
	TermMappings map[string]string     `json:"termMappings,omitempty"`
}

type PageExportBundle struct {
	SchemaVersion     string                 `json:"schemaVersion"`
	ExportedAt        time.Time              `json:"exportedAt"`
	SourceEnvironment PageSourceEnvironment  `json:"sourceEnvironment,omitempty"`
	Page              PageBundleDefinition   `json:"page"`
	Dependencies      PageBundleDependencies `json:"dependencies"`
}

type ImportPageRequest struct {
	Bundle    PageExportBundle `json:"bundle"`
	Overwrite bool             `json:"overwrite,omitempty"`
}

type PageQueryImportResult struct {
	OriginalID  string `json:"originalId"`
	NewID       string `json:"newId,omitempty"`
	Name        string `json:"name"`
	Status      string `json:"status"` // "imported" | "skipped" | "renamed" | "overwritten" | "rebound_core"
	Reason      string `json:"reason,omitempty"`
	ContentHash string `json:"contentHash"`
}

type ImportPageReport struct {
	PageID       string                  `json:"pageId"`
	PageName     string                  `json:"pageName"`
	PageSlug     string                  `json:"pageSlug"`
	Status       string                  `json:"status"` // "created" | "overwritten" | "renamed"
	QueryResults []PageQueryImportResult `json:"queryResults"`
}

type bundleSavedQueryRow struct {
	ID           string         `db:"id"`
	TenantID     string         `db:"tenant_id"`
	Name         string         `db:"name"`
	Description  string         `db:"description"`
	SourceID     string         `db:"source_id"`
	BindingID    sql.NullString `db:"binding_id"`
	RelatedBOIDs pq.StringArray `db:"related_bo_ids"`
	ChartType    string         `db:"chart_type"`
	QueryState   []byte         `db:"query_state"`
	Tags         pq.StringArray `db:"tags"`
	IsCore       bool           `db:"is_core"`
}

func computeStateHash(stateJSON []byte) string {
	var normalized interface{}
	if err := json.Unmarshal(stateJSON, &normalized); err != nil {
		h := sha256.Sum256(stateJSON)
		return "sha256:" + hex.EncodeToString(h[:])
	}
	canonical, _ := json.Marshal(normalized)
	h := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(h[:])
}

// ExtractAllReferencedQueryIDs scans all JSON fields in a page to find any referenced query IDs.
func ExtractAllReferencedQueryIDs(page *PageStudioPage) []string {
	ids := make(map[string]bool)
	scanRawJSONForQueries(page.Layout, ids)
	scanRawJSONForQueries(page.Tabs, ids)
	scanRawJSONForQueries(page.Components, ids)
	scanRawJSONForQueries(page.DataSources, ids)
	scanRawJSONForQueries(page.PresentationEvents, ids)
	scanRawJSONForQueries(page.FilterBar, ids)
	if page.App != nil {
		scanRawJSONForQueries(*page.App, ids)
	}

	result := make([]string, 0, len(ids))
	for id := range ids {
		if strings.TrimSpace(id) != "" {
			result = append(result, id)
		}
	}
	return result
}

func scanRawJSONForQueries(raw []byte, out map[string]bool) {
	if len(raw) == 0 {
		return
	}
	var val interface{}
	if err := json.Unmarshal(raw, &val); err != nil {
		return
	}
	scanInterfaceForQueries(val, out)
}

func scanInterfaceForQueries(v interface{}, out map[string]bool) {
	if v == nil {
		return
	}
	switch val := v.(type) {
	case map[string]interface{}:
		if sqID, ok := val["savedQueryId"].(string); ok && sqID != "" {
			out[sqID] = true
		}
		if qID, ok := val["queryId"].(string); ok && qID != "" {
			out[qID] = true
		}
		if target, ok := val["target"].(string); ok && target != "" {
			if strings.HasPrefix(target, "/saved-queries/") {
				out[strings.TrimPrefix(target, "/saved-queries/")] = true
			} else if strings.HasPrefix(target, "query:") {
				out[strings.TrimPrefix(target, "query:")] = true
			}
		}
		for _, child := range val {
			scanInterfaceForQueries(child, out)
		}
	case []interface{}:
		for _, item := range val {
			scanInterfaceForQueries(item, out)
		}
	}
}

// RemapPageQueryIDs replaces any occurrence of old query IDs with mapped new query IDs.
func RemapPageQueryIDs(page *PageBundleDefinition, idMap map[string]string) {
	if len(idMap) == 0 {
		return
	}
	page.Layout = remapRawJSON(page.Layout, idMap)
	page.Tabs = remapRawJSON(page.Tabs, idMap)
	page.Components = remapRawJSON(page.Components, idMap)
	page.DataSources = remapRawJSON(page.DataSources, idMap)
	page.PresentationEvents = remapRawJSON(page.PresentationEvents, idMap)
	page.FilterBar = remapRawJSON(page.FilterBar, idMap)
	if page.App != nil {
		remapped := json.RawMessage(remapRawJSON(*page.App, idMap))
		page.App = &remapped
	}
}

func remapRawJSON(raw []byte, idMap map[string]string) []byte {
	if len(raw) == 0 {
		return raw
	}
	var val interface{}
	if err := json.Unmarshal(raw, &val); err != nil {
		return raw
	}
	remappedVal := remapInterface(val, idMap)
	out, err := json.Marshal(remappedVal)
	if err != nil {
		return raw
	}
	return out
}

func remapInterface(v interface{}, idMap map[string]string) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{}, len(val))
		for k, child := range val {
			if k == "savedQueryId" || k == "queryId" {
				if oldID, ok := child.(string); ok {
					if newID, found := idMap[oldID]; found {
						newMap[k] = newID
						continue
					}
				}
			}
			if k == "target" {
				if oldTarget, ok := child.(string); ok {
					if strings.HasPrefix(oldTarget, "/saved-queries/") {
						oldID := strings.TrimPrefix(oldTarget, "/saved-queries/")
						if newID, found := idMap[oldID]; found {
							newMap[k] = "/saved-queries/" + newID
							continue
						}
					} else if strings.HasPrefix(oldTarget, "query:") {
						oldID := strings.TrimPrefix(oldTarget, "query:")
						if newID, found := idMap[oldID]; found {
							newMap[k] = "query:" + newID
							continue
						}
					} else if newID, found := idMap[oldTarget]; found {
						newMap[k] = newID
						continue
					}
				}
			}
			newMap[k] = remapInterface(child, idMap)
		}
		return newMap
	case []interface{}:
		newArr := make([]interface{}, len(val))
		for i, item := range val {
			if strVal, ok := item.(string); ok {
				if newID, found := idMap[strVal]; found {
					newArr[i] = newID
					continue
				}
			}
			newArr[i] = remapInterface(item, idMap)
		}
		return newArr
	case string:
		if newID, found := idMap[val]; found {
			return newID
		}
		return val
	default:
		return v
	}
}

// exportPage handles GET /api/page-studio/pages/{id}/export.
func (h *PageStudioHandler) exportPage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid page id", http.StatusBadRequest)
		return
	}

	page, err := h.getOne(r, "id = $1 AND tenant_id = $2", id, tenantID)
	if err == sql.ErrNoRows {
		// Fallback to core page from gold copy
		goldID := h.goldCopyID(r.Context())
		if goldID != uuid.Nil {
			page, err = h.getOne(r, "id = $1 AND is_core = true AND tenant_id = $2", id, goldID)
		}
	}
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "page not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load page: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Find all query IDs referenced inside the page
	queryIDs := ExtractAllReferencedQueryIDs(page)
	queries := make([]PageBundleQueryItem, 0, len(queryIDs))
	termMappings := make(map[string]string)

	for _, qID := range queryIDs {
		var row bundleSavedQueryRow
		err := h.db.GetContext(r.Context(), &row, `
			SELECT id, tenant_id, name, description, source_id, binding_id, related_bo_ids,
			       chart_type, query_state, tags, is_core
			FROM data_explorer.saved_query
			WHERE id = $1 AND (tenant_id = $2 OR is_core = true)
		`, qID, tenantID)
		if err != nil {
			// Query might have been deleted or missing; skip gracefully or record
			continue
		}

		hash := computeStateHash(row.QueryState)
		item := PageBundleQueryItem{
			ID:           row.ID,
			Name:         row.Name,
			Description:  row.Description,
			BOID:         row.SourceID,
			BindingID:    row.BindingID.String,
			RelatedBOIDs: []string(row.RelatedBOIDs),
			ChartType:    row.ChartType,
			State:        row.QueryState,
			Tags:         []string(row.Tags),
			Provenance: PageBundleProvenance{
				IsCore:      row.IsCore,
				OriginalID:  row.ID,
				ContentHash: hash,
			},
			Dependencies: PageBundleQueryDependencies{
				BOID:         row.SourceID,
				RelatedBOIDs: []string(row.RelatedBOIDs),
			},
		}

		// Extract term IDs from state
		var stateObj struct {
			Dimensions []struct {
				TermNodeID string `json:"termNodeId"`
			} `json:"dimensions"`
			Measures []struct {
				TermNodeID string `json:"termNodeId"`
			} `json:"measures"`
			Filters []struct {
				TermNodeID string `json:"termNodeId"`
			} `json:"filters"`
		}
		if err := json.Unmarshal(row.QueryState, &stateObj); err == nil {
			termSet := make(map[string]bool)
			for _, d := range stateObj.Dimensions {
				if d.TermNodeID != "" {
					termSet[d.TermNodeID] = true
				}
			}
			for _, m := range stateObj.Measures {
				if m.TermNodeID != "" {
					termSet[m.TermNodeID] = true
				}
			}
			for _, f := range stateObj.Filters {
				if f.TermNodeID != "" {
					termSet[f.TermNodeID] = true
				}
			}
			for t := range termSet {
				item.Dependencies.TermNodeIDs = append(item.Dependencies.TermNodeIDs, t)
				termMappings[t] = fmt.Sprintf("%s.%s", row.SourceID, t)
			}
		}

		queries = append(queries, item)
	}

	bundle := PageExportBundle{
		SchemaVersion: PageBundleSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		SourceEnvironment: PageSourceEnvironment{
			TenantID: tenantID.String(),
		},
		Page: PageBundleDefinition{
			Name:               page.Name,
			Slug:               page.Slug,
			Description:        page.Description,
			Layout:             page.Layout,
			Tabs:               page.Tabs,
			Components:         page.Components,
			DataSources:        page.DataSources,
			PresentationEvents: page.PresentationEvents,
			FilterBar:          page.FilterBar,
			App:                page.App,
		},
		Dependencies: PageBundleDependencies{
			Queries:      queries,
			TermMappings: termMappings,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.page-bundle.json"`, page.Slug))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(bundle)
}

// importPage handles POST /api/page-studio/pages/import.
func (h *PageStudioHandler) importPage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}

	var req ImportPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Bundle.SchemaVersion != PageBundleSchemaVersion {
		http.Error(w, fmt.Sprintf("unsupported schema version %q; expected %q", req.Bundle.SchemaVersion, PageBundleSchemaVersion), http.StatusUnprocessableEntity)
		return
	}

	goldID := h.goldCopyID(r.Context())
	isMaster := goldID != uuid.Nil && goldID == tenantID

	// 1. Process and import queries
	idMap := make(map[string]string)
	queryResults := make([]PageQueryImportResult, 0, len(req.Bundle.Dependencies.Queries))

	for _, item := range req.Bundle.Dependencies.Queries {
		itemHash := item.Provenance.ContentHash
		if itemHash == "" {
			itemHash = computeStateHash(item.State)
		}

		// (a) Dedupe check in target tenant: check for matching hash or exact query
		var existingID string
		err := h.db.GetContext(r.Context(), &existingID, `
			SELECT id FROM data_explorer.saved_query 
			WHERE tenant_id = $1 AND archived_at IS NULL AND source_id = $2 AND chart_type = $3 AND query_state::text = $4::text
			LIMIT 1
		`, tenantID, item.BOID, item.ChartType, string(item.State))
		if err == nil && existingID != "" {
			idMap[item.ID] = existingID
			queryResults = append(queryResults, PageQueryImportResult{
				OriginalID:  item.ID,
				NewID:       existingID,
				Name:        item.Name,
				Status:      "skipped",
				Reason:      "already-present with identical content",
				ContentHash: itemHash,
			})
			continue
		}

		// (b) Core adoption / rebind check: if client tenant and isCore, check if master tenant has matching core query
		if !isMaster && item.Provenance.IsCore && goldID != uuid.Nil {
			var coreID string
			errCore := h.db.GetContext(r.Context(), &coreID, `
				SELECT id FROM data_explorer.saved_query 
				WHERE tenant_id = $1 AND is_core = true AND (id = $2 OR name = $3) AND archived_at IS NULL
				LIMIT 1
			`, goldID, item.ID, item.Name)
			if errCore == nil && coreID != "" {
				idMap[item.ID] = coreID
				queryResults = append(queryResults, PageQueryImportResult{
					OriginalID:  item.ID,
					NewID:       coreID,
					Name:        item.Name,
					Status:      "rebound_core",
					Reason:      "rebound to gold copy core query",
					ContentHash: itemHash,
				})
				continue
			}
		}

		// Client tenant cannot create is_core = true query
		queryIsCore := item.Provenance.IsCore && isMaster

		// (c) Name collision check
		var existingSameName struct {
			ID     string `db:"id"`
			IsCore bool   `db:"is_core"`
		}
		errName := h.db.GetContext(r.Context(), &existingSameName, `
			SELECT id, is_core FROM data_explorer.saved_query 
			WHERE tenant_id = $1 AND name = $2 AND archived_at IS NULL
			LIMIT 1
		`, tenantID, item.Name)

		if errName == nil && existingSameName.ID != "" {
			if req.Overwrite {
				// Overwrite in place
				if existingSameName.IsCore && !isMaster {
					http.Error(w, fmt.Sprintf("cannot overwrite adopted core query %q", item.Name), http.StatusConflict)
					return
				}
				_, errUpd := h.db.ExecContext(r.Context(), `
					UPDATE data_explorer.saved_query
					SET description = $1, source_id = $2, binding_id = $3, related_bo_ids = $4,
					    chart_type = $5, query_state = $6, tags = $7, updated_at = NOW()
					WHERE id = $8 AND tenant_id = $9
				`, item.Description, item.BOID, sql.NullString{String: item.BindingID, Valid: item.BindingID != ""},
					pq.Array(item.RelatedBOIDs), item.ChartType, item.State, pq.Array(item.Tags),
					existingSameName.ID, tenantID)
				if errUpd != nil {
					http.Error(w, "failed to overwrite query: "+errUpd.Error(), http.StatusInternalServerError)
					return
				}
				idMap[item.ID] = existingSameName.ID
				queryResults = append(queryResults, PageQueryImportResult{
					OriginalID:  item.ID,
					NewID:       existingSameName.ID,
					Name:        item.Name,
					Status:      "overwritten",
					ContentHash: itemHash,
				})
				continue
			} else {
				// Rename query
				importedName := item.Name + " (imported)"
				newID := uuid.NewString()
				_, errIns := h.db.ExecContext(r.Context(), `
					INSERT INTO data_explorer.saved_query
					(id, tenant_id, user_id, name, description, source_id, binding_id, related_bo_ids,
					 chart_type, query_state, tags, visibility, is_core, status, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'shared', $12, 'active', NOW(), NOW())
				`, newID, tenantID, tenantID.String(), importedName, item.Description, item.BOID,
					sql.NullString{String: item.BindingID, Valid: item.BindingID != ""}, pq.Array(item.RelatedBOIDs),
					item.ChartType, item.State, pq.Array(item.Tags), queryIsCore)
				if errIns != nil {
					http.Error(w, "failed to insert renamed query: "+errIns.Error(), http.StatusInternalServerError)
					return
				}
				idMap[item.ID] = newID
				queryResults = append(queryResults, PageQueryImportResult{
					OriginalID:  item.ID,
					NewID:       newID,
					Name:        importedName,
					Status:      "renamed",
					ContentHash: itemHash,
				})
				continue
			}
		}

		// Fresh query insertion
		newID := uuid.NewString()
		_, errIns := h.db.ExecContext(r.Context(), `
			INSERT INTO data_explorer.saved_query
			(id, tenant_id, user_id, name, description, source_id, binding_id, related_bo_ids,
			 chart_type, query_state, tags, visibility, is_core, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'shared', $12, 'active', NOW(), NOW())
		`, newID, tenantID, tenantID.String(), item.Name, item.Description, item.BOID,
			sql.NullString{String: item.BindingID, Valid: item.BindingID != ""}, pq.Array(item.RelatedBOIDs),
			item.ChartType, item.State, pq.Array(item.Tags), queryIsCore)
		if errIns != nil {
			http.Error(w, "failed to insert query: "+errIns.Error(), http.StatusInternalServerError)
			return
		}
		idMap[item.ID] = newID
		queryResults = append(queryResults, PageQueryImportResult{
			OriginalID:  item.ID,
			NewID:       newID,
			Name:        item.Name,
			Status:      "imported",
			ContentHash: itemHash,
		})
	}

	// 2. Remap page query IDs
	pageDef := req.Bundle.Page
	RemapPageQueryIDs(&pageDef, idMap)

	// 3. Save Page
	pageName := pageDef.Name
	pageSlug := pageDef.Slug
	pageStatus := "created"

	var existingPage struct {
		ID uuid.UUID `db:"id"`
	}
	errPage := h.db.GetContext(r.Context(), &existingPage, `
		SELECT id FROM page_definitions WHERE tenant_id = $1 AND (slug = $2 OR name = $3)
		LIMIT 1
	`, tenantID, pageSlug, pageName)

	var targetPageID uuid.UUID

	if errPage == nil && existingPage.ID != uuid.Nil {
		if req.Overwrite {
			targetPageID = existingPage.ID
			_, errUpd := h.db.ExecContext(r.Context(), `
				UPDATE page_definitions
				SET name = $1, description = $2, layout = $3, tabs = $4, components = $5,
				    data_sources = $6, presentation_events = $7, filter_bar = $8, app_model = $9,
				    updated_at = NOW()
				WHERE id = $10 AND tenant_id = $11
			`, pageName, pageDef.Description, pageDef.Layout, pageDef.Tabs, pageDef.Components,
				pageDef.DataSources, pageDef.PresentationEvents, pageDef.FilterBar, pageDef.App,
				targetPageID, tenantID)
			if errUpd != nil {
				http.Error(w, "failed to update page: "+errUpd.Error(), http.StatusInternalServerError)
				return
			}
			pageStatus = "overwritten"
		} else {
			pageName = pageName + " (imported)"
			pageSlug = pageSlug + "-imported-" + uuid.New().String()[:8]
			targetPageID = uuid.New()
			_, errIns := h.db.ExecContext(r.Context(), `
				INSERT INTO page_definitions
				(id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
				 presentation_events, filter_bar, app_model, version, is_core, status, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 1, false, 'draft', NOW(), NOW())
			`, targetPageID, tenantID, pageName, pageSlug, pageDef.Description, pageDef.Layout,
				pageDef.Tabs, pageDef.Components, pageDef.DataSources, pageDef.PresentationEvents,
				pageDef.FilterBar, pageDef.App)
			if errIns != nil {
				http.Error(w, "failed to insert renamed page: "+errIns.Error(), http.StatusInternalServerError)
				return
			}
			pageStatus = "renamed"
		}
	} else {
		targetPageID = uuid.New()
		_, errIns := h.db.ExecContext(r.Context(), `
			INSERT INTO page_definitions
			(id, tenant_id, name, slug, description, layout, tabs, components, data_sources,
			 presentation_events, filter_bar, app_model, version, is_core, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 1, false, 'draft', NOW(), NOW())
		`, targetPageID, tenantID, pageName, pageSlug, pageDef.Description, pageDef.Layout,
			pageDef.Tabs, pageDef.Components, pageDef.DataSources, pageDef.PresentationEvents,
			pageDef.FilterBar, pageDef.App)
		if errIns != nil {
			http.Error(w, "failed to insert page: "+errIns.Error(), http.StatusInternalServerError)
			return
		}
	}

	report := ImportPageReport{
		PageID:       targetPageID.String(),
		PageName:     pageName,
		PageSlug:     pageSlug,
		Status:       pageStatus,
		QueryResults: queryResults,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(report)
}
