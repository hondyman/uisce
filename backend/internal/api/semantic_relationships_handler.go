package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/logging"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// SemanticRelationshipsHandler handles API requests for term relationships, differentiators, and AI prompt context.
type SemanticRelationshipsHandler struct {
	relService *analytics.TermRelationshipService
	db         *sqlx.DB
}

// NewSemanticRelationshipsHandler creates a new instance of SemanticRelationshipsHandler.
func NewSemanticRelationshipsHandler(relService *analytics.TermRelationshipService, db *sqlx.DB) *SemanticRelationshipsHandler {
	return &SemanticRelationshipsHandler{
		relService: relService,
		db:         db,
	}
}

// RegisterRoutes registers HTTP routes with Chi.
func (h *SemanticRelationshipsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/semantic-terms/{id}/related", h.GetRelatedTerms)
	r.Get("/business-terms/{id}/related", h.GetRelatedTerms)
	r.Get("/semantic-mapper/suggest-related", h.SuggestRelatedForColumn)
	r.Post("/semantic-mapper/ai-context", h.GetAIContext)
	r.Post("/semantic-terms/relationships", h.CreateRelationshipEdge)
	r.Post("/semantic-mapper/rejections", h.RecordRejection)
	r.Get("/semantic-mapper/rejections", h.ListRejections)
	r.Delete("/semantic-mapper/rejections/{id}", h.DeleteRejection)
	r.Get("/taxonomy/l3-classifications", h.ListL3Classifications)
	r.Get("/taxonomy/suggest-l3", h.SuggestL3Classification)
	r.Post("/taxonomy/classify-term", h.ClassifyTerm)
	r.Post("/catalog/nodes/{id}/visualize-lens", h.VisualizeLens)
}

// GetRelatedTerms returns related terms, peers, specializations, and differentiator reasoning for a term.
func (h *SemanticRelationshipsHandler) GetRelatedTerms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	termIDOrName := chi.URLParam(r, "id")
	if termIDOrName == "" {
		termIDOrName = r.URL.Query().Get("term")
	}
	if strings.TrimSpace(termIDOrName) == "" {
		http.Error(w, `{"error":"term id or name is required"}`, http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	disambiguation, err := h.relService.GetRelatedTerms(r.Context(), tenantID, datasourceID, termIDOrName)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to fetch related terms for %s: %v", termIDOrName, err)
		http.Error(w, `{"error":"failed to fetch related terms"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(disambiguation)
}

// SuggestRelatedForColumn returns candidate business terms and differentiators for a given column/table.
func (h *SemanticRelationshipsHandler) SuggestRelatedForColumn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	columnName := r.URL.Query().Get("column")
	entityName := r.URL.Query().Get("entity")
	if columnName == "" {
		columnName = r.URL.Query().Get("column_name")
	}
	if entityName == "" {
		entityName = r.URL.Query().Get("table")
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	suggestions, err := h.relService.SuggestRelatedTermsForColumn(r.Context(), tenantID, datasourceID, columnName, entityName)
	if err != nil {
		http.Error(w, `{"error":"failed to generate related suggestions"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"column":      columnName,
		"entity":      entityName,
		"suggestions": suggestions,
	})
}

// GetAIContext exports prompt-optimized semantic context and JSON-LD schema for AI agents.
func (h *SemanticRelationshipsHandler) GetAIContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		TermIDs []string `json:"term_ids"`
		Domain  string   `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is allowed; will export all core terms
		req.TermIDs = nil
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	payload, err := h.relService.BuildAIContext(r.Context(), tenantID, datasourceID, req.TermIDs, req.Domain)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to build AI context: %v", err)
		http.Error(w, `{"error":"failed to build AI context"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(payload)
}

// CreateRelationshipEdge creates a relationship edge between two terms in the catalog graph.
func (h *SemanticRelationshipsHandler) CreateRelationshipEdge(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		SourceTermID string                 `json:"source_term_id"`
		TargetTermID string                 `json:"target_term_id"`
		EdgeTypeName string                 `json:"edge_type_name"`
		Properties   map[string]interface{} `json:"properties"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.SourceTermID == "" || req.TargetTermID == "" || req.EdgeTypeName == "" {
		http.Error(w, `{"error":"source_term_id, target_term_id, and edge_type_name are required"}`, http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	edgeID, err := h.relService.CreateTermRelationship(
		r.Context(),
		tenantID,
		datasourceID,
		req.SourceTermID,
		req.TargetTermID,
		req.EdgeTypeName,
		req.Properties,
	)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create relationship edge: %v", err)
		http.Error(w, `{"error":"failed to create relationship edge"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"edge_id": edgeID,
		"status":  "created",
	})
}

// RecordRejection records a user or system rejected mapping into catalog_edge_rejection_store.
func (h *SemanticRelationshipsHandler) RecordRejection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		SourceNodeID     string `json:"source_node_id"`
		RejectedTargetID string `json:"rejected_target_id"`
		EdgeTypeID       string `json:"edge_type_id"`
		Reason           string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.SourceNodeID == "" || req.RejectedTargetID == "" {
		http.Error(w, `{"error":"source_node_id and rejected_target_id are required"}`, http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	rejectedBy := "user"
	if claims != nil {
		tenantID = claims.TenantID
		if claims.Email != "" {
			rejectedBy = claims.Email
		}
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	err := h.relService.RecordRejection(
		r.Context(),
		tenantID,
		req.SourceNodeID,
		req.RejectedTargetID,
		req.EdgeTypeID,
		rejectedBy,
		req.Reason,
	)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to record rejection: %v", err)
		http.Error(w, `{"error":"failed to record rejection"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "recorded",
	})
}

// ListRejections returns all active rejections for the tenant.
func (h *SemanticRelationshipsHandler) ListRejections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	records, err := h.relService.ListRejections(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":"failed to list rejections"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": records,
	})
}

// DeleteRejection deletes a rejection entry by ID.
func (h *SemanticRelationshipsHandler) DeleteRejection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rejectionID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	err := h.relService.DeleteRejection(r.Context(), tenantID, rejectionID)
	if err != nil {
		http.Error(w, `{"error":"failed to delete rejection"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
	})
}

// ListL3Classifications returns all 16 canonical L3 classifications with L1/L2 breadcrumbs.
func (h *SemanticRelationshipsHandler) ListL3Classifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	classifications, err := h.relService.ListL3Classifications(r.Context(), tenantID)
	if err != nil {
		http.Error(w, `{"error":"failed to list classifications"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  classifications,
		"count": len(classifications),
	})
}

// SuggestL3Classification maps input terms or columns to their optimal L3 taxonomy node.
func (h *SemanticRelationshipsHandler) SuggestL3Classification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	term := r.URL.Query().Get("term")
	column := r.URL.Query().Get("column")

	suggestion := h.relService.SuggestL3Classification(term, column)
	json.NewEncoder(w).Encode(suggestion)
}

// ClassifyTerm links a business term to an L3 classification node via CLASSIFIED_BY.
func (h *SemanticRelationshipsHandler) ClassifyTerm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		TermID   string `json:"term_id"`
		L3NodeID string `json:"l3_node_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	if req.TermID == "" || req.L3NodeID == "" {
		http.Error(w, `{"error":"term_id and l3_node_id are required"}`, http.StatusBadRequest)
		return
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := ""
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	err := h.relService.ClassifyTerm(r.Context(), tenantID, req.TermID, req.L3NodeID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to classify term: %v", err)
		http.Error(w, `{"error":"failed to classify term"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "classified",
	})
}

// VisualizeLens returns the multi-lens projection graph for the Cognitive Studio.
func (h *SemanticRelationshipsHandler) VisualizeLens(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := chi.URLParam(r, "id")
	if nodeID == "" {
		http.Error(w, `{"error":"node id is required"}`, http.StatusBadRequest)
		return
	}

	var req analytics.VisualizeLensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.LensType = analytics.LensSubtypeAndPeers
	}

	claims := jwtmiddleware.GetClaimsFromContext(r)
	tenantID := req.TenantID
	if tenantID == "" && claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}

	graphData, err := h.relService.VisualizeLens(r.Context(), tenantID, nodeID, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to build lens visualization for %s: %v", nodeID, err)
		http.Error(w, `{"error":"failed to generate lens graph"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(graphData)
}
