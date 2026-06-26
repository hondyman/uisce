package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// RelationshipHandler handles relationship inference and management API endpoints
type RelationshipHandler struct {
	inferenceService *analytics.RelationshipInferenceService
}

// NewRelationshipHandler creates a new relationship handler
func NewRelationshipHandler(svc *analytics.RelationshipInferenceService) *RelationshipHandler {
	return &RelationshipHandler{inferenceService: svc}
}

// RegisterRelationshipRoutes registers all relationship-related routes
func RegisterRelationshipRoutes(r chi.Router, handler *RelationshipHandler) {
	// Discovery and inference
	r.Post("/infer", handler.InferRelationships)
	r.Get("/physical/{tableId}", handler.GetTableRelationships)

	// Physical relationship management
	r.Post("/physical", handler.CreateTableRelationship)

	// BO relationship inheritance
	r.Post("/bo/inherit", handler.InheritBORelationships)
	r.Get("/bo/{boId}", handler.GetBORelationships)

}

// ============================================================================
// Request/Response Types
// ============================================================================

// InferRelationshipsRequest is the request body for POST /relationships/infer
type InferRelationshipsRequest struct {
	TableIDs []string `json:"table_ids"`
}

// InferRelationshipsResponse is the response for POST /relationships/infer
type InferRelationshipsResponse struct {
	Candidates []analytics.RelationshipCandidate `json:"candidates"`
	Count      int                               `json:"count"`
}

// CreateTableRelationshipRequest is the request for POST /relationships/physical
type CreateTableRelationshipRequest struct {
	SourceTableID   string  `json:"source_table_id"`
	TargetTableID   string  `json:"target_table_id"`
	JoinCondition   string  `json:"join_condition"`
	JoinType        string  `json:"join_type"`
	Cardinality     string  `json:"cardinality"`
	Confidence      float64 `json:"confidence"`
	Origin          string  `json:"origin"`
	LookupCandidate bool    `json:"lookup_candidate"`
	Notes           string  `json:"notes,omitempty"`
}

// InheritBORelationshipsRequest is the request for POST /relationships/bo/inherit
type InheritBORelationshipsRequest struct {
	BONodeID           string `json:"bo_node_id"`
	DrivingTableNodeID string `json:"driving_table_node_id"`
}

// ============================================================================
// Handlers
// ============================================================================

// InferRelationships discovers candidate relationships between tables
// POST /api/relationships/infer
func (h *RelationshipHandler) InferRelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant context
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant or datasource ID", http.StatusBadRequest)
		return
	}

	datasourceUUID, err := uuid.Parse(datasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req InferRelationshipsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert table IDs to UUIDs
	tableUUIDs := make([]uuid.UUID, len(req.TableIDs))
	for i, id := range req.TableIDs {
		tableUUID, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid table ID: "+id, http.StatusBadRequest)
			return
		}
		tableUUIDs[i] = tableUUID
	}

	// Discover relationships
	candidates, err := h.inferenceService.DiscoverTableRelationships(ctx, datasourceUUID, tableUUIDs)
	if err != nil {
		http.Error(w, "Failed to discover relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := InferRelationshipsResponse{
		Candidates: candidates,
		Count:      len(candidates),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetTableRelationships retrieves physical relationships for a table
// GET /api/relationships/physical/{tableId}
func (h *RelationshipHandler) GetTableRelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		http.Error(w, "Missing datasource ID", http.StatusBadRequest)
		return
	}

	datasourceUUID, err := uuid.Parse(datasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource ID", http.StatusBadRequest)
		return
	}

	tableID := chi.URLParam(r, "tableId")
	tableUUID, err := uuid.Parse(tableID)
	if err != nil {
		http.Error(w, "Invalid table ID", http.StatusBadRequest)
		return
	}

	edges, err := h.inferenceService.GetTableRelationships(ctx, datasourceUUID, tableUUID)
	if err != nil {
		http.Error(w, "Failed to get relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edges)
}

// CreateTableRelationship creates a TABLE_RELATES_TO_TABLE edge
// POST /api/relationships/physical
func (h *RelationshipHandler) CreateTableRelationship(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant or datasource ID", http.StatusBadRequest)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	datasourceUUID, err := uuid.Parse(datasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource ID", http.StatusBadRequest)
		return
	}

	var req CreateTableRelationshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sourceUUID, err := uuid.Parse(req.SourceTableID)
	if err != nil {
		http.Error(w, "Invalid source table ID", http.StatusBadRequest)
		return
	}

	targetUUID, err := uuid.Parse(req.TargetTableID)
	if err != nil {
		http.Error(w, "Invalid target table ID", http.StatusBadRequest)
		return
	}

	edge := analytics.TableRelationshipEdge{
		SourceNodeID:    sourceUUID,
		TargetNodeID:    targetUUID,
		JoinCondition:   req.JoinCondition,
		JoinType:        req.JoinType,
		Cardinality:     req.Cardinality,
		Confidence:      req.Confidence,
		Origin:          req.Origin,
		LookupCandidate: req.LookupCandidate,
		Notes:           req.Notes,
	}

	edgeID, err := h.inferenceService.CreateTableRelationship(ctx, tenantUUID, datasourceUUID, edge)
	if err != nil {
		http.Error(w, "Failed to create relationship: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      edgeID.String(),
		"success": true,
	})
}

// InheritBORelationships creates BO_RELATES_TO_BO edges from physical table relationships
// POST /api/relationships/bo/inherit
func (h *RelationshipHandler) InheritBORelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if tenantID == "" || datasourceID == "" {
		http.Error(w, "Missing tenant or datasource ID", http.StatusBadRequest)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	datasourceUUID, err := uuid.Parse(datasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource ID", http.StatusBadRequest)
		return
	}

	var req InheritBORelationshipsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	boNodeUUID, err := uuid.Parse(req.BONodeID)
	if err != nil {
		http.Error(w, "Invalid BO node ID", http.StatusBadRequest)
		return
	}

	drivingTableUUID, err := uuid.Parse(req.DrivingTableNodeID)
	if err != nil {
		http.Error(w, "Invalid driving table node ID", http.StatusBadRequest)
		return
	}

	boRels, err := h.inferenceService.InheritBORelationshipsFromTable(
		ctx, tenantUUID, datasourceUUID, boNodeUUID, drivingTableUUID,
	)
	if err != nil {
		http.Error(w, "Failed to inherit relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"relationships": boRels,
		"count":         len(boRels),
	})
}

// GetBORelationships retrieves BO_RELATES_TO_BO edges for a business object
// GET /api/relationships/bo/{boId}
func (h *RelationshipHandler) GetBORelationships(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		http.Error(w, "Missing datasource ID", http.StatusBadRequest)
		return
	}

	datasourceUUID, err := uuid.Parse(datasourceID)
	if err != nil {
		http.Error(w, "Invalid datasource ID", http.StatusBadRequest)
		return
	}

	boID := chi.URLParam(r, "boId")
	boUUID, err := uuid.Parse(boID)
	if err != nil {
		http.Error(w, "Invalid BO ID", http.StatusBadRequest)
		return
	}

	edges, err := h.inferenceService.GetBORelationships(ctx, datasourceUUID, boUUID)
	if err != nil {
		http.Error(w, "Failed to get BO relationships: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edges)
}
