package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type CatalogHandler struct {
	boService *metadata.BusinessObjectService
}

func NewCatalogHandler(boService *metadata.BusinessObjectService) *CatalogHandler {
	return &CatalogHandler{boService: boService}
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
	// Some clients might pass tenant_id in query
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}

	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header or tenant_id query param is required", http.StatusBadRequest)
		return
	}

	// Datasource ID can be passed as X-Tenant-Datasource-ID or datasource_id query
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
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

// handleGetSemanticTermsByTable returns semantic terms linked to columns from a specific driver table
func (h *CatalogHandler) handleGetSemanticTermsByTable(w http.ResponseWriter, r *http.Request) {
	tableID := chi.URLParam(r, "tableId")
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tableID == "" {
		http.Error(w, "Table ID is required", http.StatusBadRequest)
		return
	}

	if datasourceID == "" {
		http.Error(w, "X-Tenant-Datasource-ID header is required", http.StatusBadRequest)
		return
	}

	terms, err := h.boService.GetSemanticTermsByTable(r.Context(), tableID, datasourceID)
	if err != nil {
		logging.GetLogger().Error(err.Error())
		http.Error(w, "Failed to fetch semantic terms", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"semanticTerms": terms,
	})
}
