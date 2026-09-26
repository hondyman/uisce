package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type CatalogHandler struct {
	boService    *metadata.BusinessObjectService
	securityDeps handlers.SecurityContextDeps
	db           *sqlx.DB // term -> column mappings for semantic-terms-by-table (nil: none)
}

// WithDB lets semantic-terms-by-table say which column each term maps to.
func (h *CatalogHandler) WithDB(db *sqlx.DB) *CatalogHandler {
	h.db = db
	return h
}

func NewCatalogHandler(boService *metadata.BusinessObjectService, securityDeps handlers.SecurityContextDeps) *CatalogHandler {
	return &CatalogHandler{boService: boService, securityDeps: securityDeps}
}

func (h *CatalogHandler) RegisterRoutes(r chi.Router) {
	r.Route("/catalog", func(r chi.Router) {
		r.Get("/business-terms/{id}", h.handleGetBusinessTerm)
		r.Put("/business-terms/{id}/compliance", h.handleUpdateCompliance)
		r.Post("/business-terms/{id}/mappings", h.handleAddMappings)
		r.Delete("/business-terms/{id}/mappings/{semId}", h.handleRemoveMapping)
		r.Get("/nodes", h.handleGetNodes)
		r.Get("/semantic-terms-by-table/{tableId}", h.handleGetSemanticTermsByTable)
	})
}

func (h *CatalogHandler) handleGetNodes(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
		return
	}

	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err != nil {
			http.Error(w, "X-Tenant-Datasource-ID header is required", http.StatusBadRequest)
			return
		}
		datasourceID = secCtx.DatasourceID
	}

	nodeType := r.URL.Query().Get("type")
	searchQuery := r.URL.Query().Get("q")

	nodes, err := h.boService.ListCatalogNodes(r.Context(), tenantID, datasourceID, nodeType, searchQuery)
	if err != nil {
		logging.GetLogger().Error(err.Error())
		http.Error(w, "Failed to list catalog nodes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func (h *CatalogHandler) handleGetBusinessTerm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	term, err := h.boService.GetBusinessTerm(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to fetch business term", http.StatusInternalServerError)
		return
	}
	if term == nil {
		http.Error(w, "Business term not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(term)
}

func (h *CatalogHandler) handleUpdateCompliance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req metadata.UpdateBusinessTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.boService.UpdateBusinessTerm(r.Context(), id, req); err != nil {
		logging.GetLogger().Error(err.Error())
		http.Error(w, "Failed to update compliance", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type AddMappingsRequest struct {
	SemanticTermIDs []string `json:"semanticTermIds"`
}

func (h *CatalogHandler) handleAddMappings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req AddMappingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.boService.AddBusinessTermMappings(r.Context(), id, req.SemanticTermIDs); err != nil {
		logging.GetLogger().Error(err.Error())
		http.Error(w, "Failed to add mappings", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *CatalogHandler) handleRemoveMapping(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	semId := chi.URLParam(r, "semId")

	if err := h.boService.RemoveBusinessTermMapping(r.Context(), id, semId); err != nil {
		logging.GetLogger().Error(err.Error())
		http.Error(w, "Failed to remove mapping", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetSemanticTermsByTable returns semantic terms linked to columns from a
// specific driver table, plus every calculated term for the tenant (calculated
// terms - term_type "calculated" - aren't linked to any physical column, so
// they'd never show up via the column-edge join alone; they're eligible for
// any business object regardless of driver table).
func (h *CatalogHandler) handleGetSemanticTermsByTable(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("tenant_instance_id")
	}

	if tableID == "" {
		http.Error(w, "Table ID is required", http.StatusBadRequest)
		return
	}

	// The tenant comes from the authenticated token only - never from a
	// request header on its own.
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil || secCtx == nil || secCtx.TenantID == "" {
		http.Error(w, "authentication with a tenant is required", http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	terms, err := h.boService.GetSemanticTermsByTable(r.Context(), tableID, datasourceID, tenantID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Warning in handleGetSemanticTermsByTable for table %s: %v", tableID, err)
		terms = []models.CatalogNode{}
	}
	if terms == nil {
		terms = []models.CatalogNode{}
	}

	// Say which column of this table each term maps to (MAPS_TO), so the
	// binding wizard can resolve fields instead of guessing from properties.
	mappings := h.termColumnMappings(r, tableID, terms)
	out := make([]termWithMappings, len(terms))
	for i, t := range terms {
		out[i] = termWithMappings{CatalogNode: t, Mappings: mappings[t.ID]}
		if out[i].Mappings == nil {
			out[i].Mappings = []termColumnMapping{}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"semanticTerms": out,
	})
}

type termColumnMapping struct {
	ColumnNodeID    string `json:"column_node_id" db:"column_node_id"`
	ColumnName      string `json:"column_name" db:"column_name"`
	TableNodeID     string `json:"table_node_id" db:"table_node_id"`
	TableName       string `json:"table_name" db:"table_name"`
	IsPrimarySource bool   `json:"is_primary_source" db:"-"`
	TermID          string `json:"-" db:"term_id"`
}

type termWithMappings struct {
	models.CatalogNode
	Mappings []termColumnMapping `json:"mappings"`
}

// termColumnMappings is term id -> the columns of tableID it MAPS_TO. The
// terms were already scoped to the caller's tenant; a mapping is only read
// for them. Errors are logged and leave terms without mappings (the wizard
// then marks them unresolved, as before).
func (h *CatalogHandler) termColumnMappings(r *http.Request, tableID string, terms []models.CatalogNode) map[string][]termColumnMapping {
	out := map[string][]termColumnMapping{}
	if h.db == nil || len(terms) == 0 {
		return out
	}
	if _, err := uuid.Parse(tableID); err != nil {
		return out
	}
	ids := make([]string, len(terms))
	for i, t := range terms {
		ids[i] = t.ID
	}
	var rows []termColumnMapping
	err := h.db.SelectContext(r.Context(), &rows, `
		SELECT ce.source_node_id::text AS term_id, col.id::text AS column_node_id, col.node_name AS column_name,
		       tbl.id::text AS table_node_id, tbl.node_name AS table_name
		FROM catalog_edge ce
		JOIN catalog_edge_type et ON et.id = ce.edge_type_id AND et.edge_type_name = 'MAPS_TO'
		JOIN catalog_node col ON col.id = ce.target_node_id
		JOIN catalog_node tbl ON tbl.id = col.parent_id
		WHERE tbl.id = $1::uuid AND ce.source_node_id::text = ANY($2)
		ORDER BY col.node_name`, tableID, pq.Array(ids))
	if err != nil {
		logging.GetLogger().Sugar().Warnf("semantic-terms-by-table %s: term mappings: %v", tableID, err)
		return out
	}
	for _, m := range rows {
		m.IsPrimarySource = len(out[m.TermID]) == 0
		out[m.TermID] = append(out[m.TermID], m)
	}
	return out
}
