package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SemanticMappingHandler handles HTTP requests for semantic mapping operations
type SemanticMappingHandler struct {
	service *analytics.SemanticMappingService
}

// NewSemanticMappingHandler creates a new semantic mapping handler
func NewSemanticMappingHandler(service *analytics.SemanticMappingService) *SemanticMappingHandler {
	return &SemanticMappingHandler{
		service: service,
	}
}

func (h *SemanticMappingHandler) RegisterRoutes(r chi.Router) {
	r.Route("/semantic-mapping", func(r chi.Router) {
		r.Post("/enrich/suggest", h.HandleSuggestEnrichment)
		r.Post("/enrich/apply", h.HandleApplyEnrichment)
		r.Post("/generate", h.HandleGenerateMappings)
		r.Post("/apply", h.HandleApplyMappings)
		r.Post("/enrich/auto", h.HandleAutoEnrichment)
		r.Post("/populate-bo-terms", h.HandlePopulateBusinessObjectSemanticTerms)

		r.Post("/backfill-sql-properties", h.HandleBackfillSemanticTermSQLProperties)
		r.Post("/backfill-physical-mappings", h.HandleBackfillPhysicalMappings)

		// Semantic Mapping Wizard
		r.Route("/wizard", func(r chi.Router) {
			r.Post("/generate", h.HandleGenerateMappingsWizard)
			r.Post("/apply", h.HandleApplyMappingsWizard)
			r.Get("/pending", h.HandleGetPendingApprovalsWizard)
			r.Get("/created", h.HandleGetCreatedMappingsWizard)
			r.Post("/approve/{id}", h.HandleApprovePendingMappingWizard)
			r.Post("/enrich-from-feedback", h.HandleEnrichTermsFromFeedback)
		})

		// Hierarchy endpoints
		r.Route("/hierarchies", func(r chi.Router) {
			r.Post("/generate", h.HandleGenerateHierarchies)
			r.Post("/create", h.HandleCreateHierarchy)
		})
	})
}

// respondJSON is a helper method to send JSON responses
func (h *SemanticMappingHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// SuggestEnrichmentRequest represents the request body for enrichment suggestion
type SuggestEnrichmentRequest struct {
	Column  analytics.DatabaseColumn  `json:"column"`
	Profile *analytics.NodeProperties `json:"profile,omitempty"`
}

// HandleSuggestEnrichment handles the enrichment suggestion request
// POST /api/semantic-mapping/enrich/suggest
func (h *SemanticMappingHandler) HandleSuggestEnrichment(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req SuggestEnrichmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Call the service
	proposal, err := h.service.SuggestEnrichment(r.Context(), &req.Column, req.Profile)
	if err != nil {
		logger.Errorf("Failed to suggest enrichment: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate suggestion"})
		return
	}

	h.respondJSON(w, http.StatusOK, proposal)
}

// HandleApplyEnrichment applies the enrichment suggestion (creates terms/edges)
// POST /api/semantic-mapping/enrich/apply
func (h *SemanticMappingHandler) HandleApplyEnrichment(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req analytics.ApplyEnrichmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body for apply enrichment: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Call the service to apply the enrichment
	ids, err := h.service.ApplyEnrichment(r.Context(), &req)
	if err != nil {
		logger.Errorf("Failed to apply enrichment: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to apply enrichment"})
		return
	}

	h.respondJSON(w, http.StatusOK, ids)
}

// AutoEnrichmentRequest represents the request body for auto enrichment
type AutoEnrichmentRequest struct {
	TenantID     string  `json:"tenant_id"`
	DatasourceID string  `json:"datasource_id"`
	Threshold    float64 `json:"threshold"`
}

// HandleAutoEnrichment triggers the auto-generation process
// POST /api/semantic-mapping/enrich/auto
func (h *SemanticMappingHandler) HandleAutoEnrichment(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req AutoEnrichmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body for auto enrichment: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.Threshold <= 0 {
		req.Threshold = 0.85 // Default threshold
	}

	result, err := h.service.AutoGenerateSemanticTerms(r.Context(), req.TenantID, req.DatasourceID, req.Threshold)
	if err != nil {
		logger.Errorf("Failed to run auto enrichment: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to run auto enrichment"})
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

// HandleGenerateMappings generates semantic suggestions for all columns
// POST /api/semantic-mapping/generate
func (h *SemanticMappingHandler) HandleGenerateMappings(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID and X-Tenant-Datasource-ID headers are required"})
		return
	}

	results, err := h.service.GenerateMappings(r.Context(), tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to generate mappings: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate mappings"})
		return
	}

	h.respondJSON(w, http.StatusOK, results)
}

// ApplyMappingsRequest request body
type ApplyMappingsRequest struct {
	Mappings []analytics.MappingResult `json:"mappings"`
}

// HandleApplyMappings applies selected mappings
// POST /api/semantic-mapping/apply
func (h *SemanticMappingHandler) HandleApplyMappings(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID and X-Tenant-Datasource-ID headers are required"})
		return
	}

	var req ApplyMappingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body for apply mappings: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	count, err := h.service.ApplyMappings(r.Context(), tenantID, datasourceID, req.Mappings)
	if err != nil {
		logger.Errorf("Failed to apply mappings: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to apply mappings"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]int{"applied_count": count})
}

// HandleGenerateMappingsWizard generates semantic mappings for all unmapped columns using AI
// POST /api/semantic-mapping/wizard/generate
func (h *SemanticMappingHandler) HandleGenerateMappingsWizard(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req analytics.GenerateMappingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	response, err := h.service.GenerateMappingsWithAI(r.Context(), &req)
	if err != nil {
		logger.Errorf("Failed to generate mappings: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate mappings"})
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleApplyMappingsWizard applies generated mappings based on confidence thresholds
// POST /api/semantic-mapping/wizard/apply
func (h *SemanticMappingHandler) HandleApplyMappingsWizard(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var request struct {
		TenantID            string                       `json:"tenant_id"`
		DatasourceID        string                       `json:"datasource_id"`
		TenantInstanceID    string                       `json:"datasource_id"`
		AutoCreateThreshold float64                      `json:"auto_create_threshold"`
		ApprovalThreshold   float64                      `json:"approval_threshold"`
		Mappings            []analytics.GeneratedMapping `json:"mappings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Handle field alias
	if request.DatasourceID == "" && request.TenantInstanceID != "" {
		request.DatasourceID = request.TenantInstanceID
	}

	if request.AutoCreateThreshold == 0 {
		request.AutoCreateThreshold = 0.85
	}
	if request.ApprovalThreshold == 0 {
		request.ApprovalThreshold = 0.60
	}

	applyReq := &analytics.ApplyMappingsRequest{
		TenantID:            request.TenantID,
		DatasourceID:        request.DatasourceID,
		TenantInstanceID:    request.TenantInstanceID,
		AutoCreateThreshold: request.AutoCreateThreshold,
		ApprovalThreshold:   request.ApprovalThreshold,
	}

	response, err := h.service.ApplyMappingsBatch(r.Context(), applyReq, request.Mappings)
	if err != nil {
		logger.Errorf("Failed to apply mappings: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to apply mappings"})
		return
	}

	h.respondJSON(w, http.StatusOK, response)
}

// HandleGetPendingApprovalsWizard retrieves pending semantic mappings for approval
// GET /api/semantic-mapping/wizard/pending?tenant_id=X&datasource_id=Y
func (h *SemanticMappingHandler) HandleGetPendingApprovalsWizard(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and datasource_id are required"})
		return
	}

	pending, err := h.service.GetPendingApprovals(r.Context(), tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to get pending approvals: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get pending approvals"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"pending_approvals": pending,
		"count":             len(pending),
	})
}

// HandleApprovePendingMappingWizard approves or rejects a pending mapping
// POST /api/semantic-mapping/wizard/approve/{id}
func (h *SemanticMappingHandler) HandleApprovePendingMappingWizard(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	mappingID := chi.URLParam(r, "id")
	if mappingID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "mapping ID is required"})
		return
	}

	var request struct {
		Approved bool   `json:"approved"`
		UserID   string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	termID, err := h.service.ApprovePendingMapping(r.Context(), mappingID, request.Approved, request.UserID)
	if err != nil {
		logger.Errorf("Failed to approve/reject mapping: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to process approval"})
		return
	}

	status := "rejected"
	if request.Approved {
		status = "approved"
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status":  status,
		"message": "Mapping " + status + " successfully",
		"term_id": termID,
	})
}

// HandleGetCreatedMappingsWizard retrieves recently created semantic mappings
// GET /api/semantic-mapping/wizard/created?tenant_id=X&datasource_id=Y&limit=N
func (h *SemanticMappingHandler) HandleGetCreatedMappingsWizard(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	limit := r.URL.Query().Get("limit")

	if tenantID == "" || datasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and datasource_id are required"})
		return
	}

	limitInt := 20
	if limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 {
			limitInt = parsed
		}
	}

	type CreatedMapping struct {
		ColumnName   string    `json:"column_name" db:"column_name"`
		SemanticTerm string    `json:"semantic_term" db:"semantic_term"`
		BusinessTerm string    `json:"business_term" db:"business_term"`
		MappedAt     time.Time `json:"mapped_at" db:"mapped_at"`
	}

	query := `
		SELECT 
			cn_col.node_name as column_name,
			cn_sem.node_name as semantic_term,
			cn_bus.node_name as business_term,
			ce1.created_at as mapped_at
		FROM catalog_edge ce1
		JOIN catalog_node cn_col ON ce1.source_node_id = cn_col.id
		JOIN catalog_node cn_sem ON ce1.target_node_id = cn_sem.id
		LEFT JOIN catalog_edge ce2 ON cn_sem.id = ce2.source_node_id AND ce2.edge_type_name = 'HAS_BUSINESS_TERM'
		LEFT JOIN catalog_node cn_bus ON ce2.target_node_id = cn_bus.id
		WHERE ce1.edge_type_name = 'MAPS_TO'
			AND cn_col.tenant_id = $1
			AND cn_col.tenant_datasource_id = $2
		ORDER BY ce1.created_at DESC
		LIMIT $3
	`

	var mappings []CreatedMapping
	err := h.service.DB().SelectContext(r.Context(), &mappings, query, tenantID, datasourceID, limitInt)
	if err != nil {
		logger.Errorf("Failed to fetch created mappings: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch created mappings"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"mappings": mappings,
		"count":    len(mappings),
	})
}

// HandlePopulateBusinessObjectSemanticTerms triggers the population of semantic terms for business objects based on physical mappings.
// POST /api/semantic-mapping/populate-bo-terms
func (h *SemanticMappingHandler) HandlePopulateBusinessObjectSemanticTerms(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantID == "" || datasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "X-Tenant-ID and X-Tenant-Datasource-ID headers are required"})
		return
	}

	count, err := h.service.PopulateBusinessObjectSemanticTerms(r.Context(), tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to populate BO semantic terms: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to populate BO semantic terms"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"populated_count": count,
		"status":          "success",
		"message":         "Successfully populated semantic terms for business objects",
	})
}

// HandleBackfillSemanticTermSQLProperties backfills SQL properties for existing semantic terms
// POST /api/semantic-mapping/backfill-sql-properties
func (h *SemanticMappingHandler) HandleBackfillSemanticTermSQLProperties(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	backfillAll := r.URL.Query().Get("all") == "true"

	if backfillAll {
		// Backfill for all tenants/datasources
		logger.Info("Starting SQL property backfill for all tenants")
		results, err := h.service.BackfillAllTenantsSemanticTermSQLProperties(r.Context())
		if err != nil {
			logger.Errorf("Failed to backfill SQL properties for all tenants: %v", err)
			h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to backfill SQL properties"})
			return
		}

		totalCount := 0
		for _, count := range results {
			totalCount += count
		}

		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":        "success",
			"total_updated": totalCount,
			"details":       results,
			"message":       "Successfully backfilled SQL properties for semantic terms",
		})
		return
	}

	// Backfill for specific tenant/datasource
	if tenantID == "" || datasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and datasource_id query parameters are required, or use all=true"})
		return
	}

	logger.Infof("Starting SQL property backfill for tenant %s, datasource %s", tenantID, datasourceID)
	count, err := h.service.BackfillSemanticTermSQLProperties(r.Context(), tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to backfill SQL properties: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to backfill SQL properties"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"updated_count": count,
		"message":       "Successfully backfilled SQL properties for semantic terms",
	})
}

// HandleBackfillPhysicalMappings backfills physical_mapping properties for existing semantic terms
// POST /api/semantic-mapping/backfill-physical-mappings
func (h *SemanticMappingHandler) HandleBackfillPhysicalMappings(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	backfillAll := r.URL.Query().Get("all") == "true"

	if backfillAll {
		// Backfill for all tenants/datasources
		logger.Info("Starting physical_mapping backfill for all tenants")
		results, err := h.service.BackfillAllTenantsPhysicalMappings(r.Context())
		if err != nil {
			logger.Errorf("Failed to backfill physical_mapping for all tenants: %v", err)
			h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to backfill physical_mapping"})
			return
		}

		totalCount := 0
		for _, count := range results {
			totalCount += count
		}

		h.respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":        "success",
			"total_updated": totalCount,
			"details":       results,
			"message":       "Successfully backfilled physical_mapping for semantic terms",
		})
		return
	}

	// Backfill for specific tenant/datasource
	if tenantID == "" || datasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and datasource_id query parameters are required, or use all=true"})
		return
	}

	logger.Infof("Starting physical_mapping backfill for tenant %s, datasource %s", tenantID, datasourceID)
	count, err := h.service.BackfillPhysicalMappings(r.Context(), tenantID, datasourceID)
	if err != nil {
		logger.Errorf("Failed to backfill physical_mapping: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to backfill physical_mapping"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"updated_count": count,
		"message":       "Successfully backfilled physical_mapping for semantic terms",
	})
}

// HandleGenerateHierarchies triggers AI-based hierarchy generation
// POST /api/semantic-mapping/hierarchies/generate
func (h *SemanticMappingHandler) HandleGenerateHierarchies(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req struct {
		TenantID     string `json:"tenant_id"`
		DatasourceID string `json:"datasource_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.TenantID == "" || req.DatasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and datasource_id are required"})
		return
	}

	hierarchies, err := h.service.GenerateHierarchiesWithAI(r.Context(), req.TenantID, req.DatasourceID)
	if err != nil {
		logger.Errorf("Failed to generate hierarchies: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate hierarchies"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"hierarchies": hierarchies,
		"count":       len(hierarchies),
	})
}

// HandleCreateHierarchy creates a new semantic hierarchy
// POST /api/semantic-mapping/hierarchies/create
func (h *SemanticMappingHandler) HandleCreateHierarchy(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()

	var req struct {
		TenantID     string                       `json:"tenant_id"`
		DatasourceID string                       `json:"datasource_id"`
		Hierarchy    analytics.GeneratedHierarchy `json:"hierarchy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.TenantID == "" || req.DatasourceID == "" {
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and datasource_id are required"})
		return
	}

	id, err := h.service.CreateHierarchy(r.Context(), req.TenantID, req.DatasourceID, req.Hierarchy)
	if err != nil {
		logger.Errorf("Failed to create hierarchy: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create hierarchy"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"hierarchy_id": id,
		"message":      "Hierarchy created successfully",
	})
}

// HandleEnrichTermsFromFeedback triggers enrichment of business terms based on feedback
// POST /api/semantic-mapping/wizard/enrich-from-feedback
func (h *SemanticMappingHandler) HandleEnrichTermsFromFeedback(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger().Sugar()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	count, err := h.service.EnrichTermsFromFeedback(r.Context(), tenantID)
	if err != nil {
		logger.Errorf("Failed to enrich terms from feedback: %v", err)
		h.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to enrich terms"})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"updated_terms": count,
		"message":       fmt.Sprintf("Enriched %d terms based on approved feedback", count),
	})
}
