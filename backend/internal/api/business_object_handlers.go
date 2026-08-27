package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	boPkg "github.com/hondyman/uisce/backend/internal/bo"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/logging"
	catalogmeta "github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)


// BOService defines the subset of BusinessObject service methods used by handlers
type BOService interface {
	GetBusinessObject(ctx context.Context, secCtx *security.Context, boKey string) (*models.BusinessObjectDefinition, error)
	GetBusinessObjectRelationships(ctx context.Context, secCtx *security.Context, boID string) (*catalogmeta.BORelationshipsResponse, error)
	ListBusinessObjects(ctx context.Context, secCtx *security.Context) ([]*models.BusinessObjectDefinition, error)
	ListBusinessObjectsComposed(ctx context.Context, secCtx *security.Context) ([]*models.BusinessObjectDefinition, error) // Workday-style Core+Custom composition
	CreateBusinessObject(ctx context.Context, secCtx *security.Context, req models.CreateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error)
	UpdateBusinessObject(ctx context.Context, secCtx *security.Context, boKey string, req models.UpdateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error)
	DeleteBusinessObject(ctx context.Context, secCtx *security.Context, boKey, userID string) error
	RenameSubtype(ctx context.Context, secCtx *security.Context, boKey, subtypeKey, newName, userID string) (*models.BusinessObjectDefinition, error)
	DeleteSubtype(ctx context.Context, secCtx *security.Context, boKey, subtypeKey, userID string) (*models.BusinessObjectDefinition, error)
	IntrospectTable(ctx context.Context, secCtx *security.Context, tableIDOrName string) (*models.TableIntrospectionResponse, error)
	QueryBORecords(ctx context.Context, secCtx *security.Context, boIDOrKey string, req models.BORecordQueryRequest) (*models.BORecordQueryResponse, error)
	CreateBORecord(ctx context.Context, secCtx *security.Context, boIDOrKey string, req models.BOCrudRecordRequest, userID string) (map[string]interface{}, error)
	UpdateBORecord(ctx context.Context, secCtx *security.Context, boIDOrKey string, recordID string, req models.BOCrudRecordRequest, userID string) (map[string]interface{}, error)
	DeleteBORecord(ctx context.Context, secCtx *security.Context, boIDOrKey string, recordID string, userID string) error
	GetBODelta(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BODeltaResponse, error)
	SynthesizeBOWithAI(ctx context.Context, secCtx *security.Context, req models.BOAISynthesizeRequest) (*models.BOAISynthesizeResponse, error)
	TranslateNLToQueryDef(ctx context.Context, secCtx *security.Context, req models.BOAINLQRequest) (*models.BOAINLQResponse, error)
	ExplainDeltaWithAI(ctx context.Context, secCtx *security.Context, req models.BOAIExplainDeltaRequest) (*models.BOAIExplainDeltaResponse, error)
	DetectAnomaliesWithAI(ctx context.Context, secCtx *security.Context, req models.BOAIAnomalyDetectRequest) (*models.BOAIAnomalyDetectResponse, error)
	GetBOWorkflowStatus(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOWorkflowStatusResponse, error)
	ExecuteWorkflowAction(ctx context.Context, secCtx *security.Context, boIDOrKey string, req models.BOWorkflowActionRequest, userID string) (*models.BOWorkflowStatusResponse, error)

	// Architectural Features 2-7 & Pillars 1-5
	DiscoverBindingScope(ctx context.Context, secCtx *security.Context, boIDOrKey string, drivingNodeID string) (*models.BOScopeDiscoveryResponse, error)
	ValidatePublishGate(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOPublishGateValidationResponse, error)
	GetMultiBackendConfiguration(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOMultiBackendConfiguration, error)
	PerformGraphRAGContext(ctx context.Context, secCtx *security.Context, req models.GraphRAGContextRequest) (*models.GraphRAGContextResponse, error)
	SimulateLineageImpact(ctx context.Context, secCtx *security.Context, req models.BOLineageImpactSimulationRequest) (*models.BOLineageImpactSimulationResponse, error)
	GenerateBOArtifacts(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BOArtifactGenerationResponse, error)
	EvaluateQueryCost(ctx context.Context, secCtx *security.Context, req models.BOQueryCostEvaluationRequest) (*models.BOQueryCostEvaluationResponse, error)
	DetectSchemaDrift(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.BODataQualitySentinelResponse, error)
	ApplyDriftRepairPatch(ctx context.Context, secCtx *security.Context, req models.BODriftRepairPatchRequest, userID string) (*models.BODriftRepairPatchResponse, error)
	RunLakehouseCompaction(ctx context.Context, secCtx *security.Context, boIDOrKey string) (*models.LakehouseMaintenanceReport, error)
}

type BusinessObjectHandler struct {
	service            BOService
	datasourceResolver security.DatasourceResolver
	db                 *sqlx.DB
	entitlements       *security.EntitlementsService
}

func NewBusinessObjectHandler(service BOService, datasourceResolver security.DatasourceResolver, sqlxDB *sqlx.DB, entSvc *security.EntitlementsService) *BusinessObjectHandler {
	return &BusinessObjectHandler{
		service:            service,
		datasourceResolver: datasourceResolver,
		db:                 sqlxDB,
		entitlements:       entSvc,
	}
}

func (h *BusinessObjectHandler) RegisterRoutes(r chi.Router) {
	r.Route("/business-objects", func(r chi.Router) {
		r.Get("/", h.ListBusinessObjects)
		r.Post("/", h.CreateBusinessObject)
		r.Post("/introspect-table", h.IntrospectTable)
		r.Post("/ai/synthesize", h.SynthesizeBOWithAI)
		r.Post("/ai/nlq", h.TranslateNLToQueryDef)
		r.Post("/ai/graph-rag", h.PerformGraphRAGContext)
		r.Post("/evaluate-cost", h.EvaluateQueryCost)
		// Subtype management routes
		r.Route("/{id}/subtypes", func(r chi.Router) {
			r.Post("/{subtypeId}/rename", h.RenameSubtype)
			r.Delete("/{subtypeId}", h.DeleteSubtype)
		})

		r.Get("/{id}/with_bindings", h.GetBusinessObjectWithBindings)
		r.Get("/{id}", h.GetBusinessObject)
		r.Get("/{id}/fields", h.GetBusinessObjectFields)
		r.Get("/{id}/relationships", h.GetBusinessObjectRelationships)
		r.Post("/{id}/relationships", h.CreateBusinessObjectRelationship)
		r.Put("/{id}/relationships/{relationshipId}", h.UpdateBusinessObjectRelationship)
		r.Delete("/{id}/relationships/{relationshipId}", h.DeleteBusinessObjectRelationship)
		r.Get("/{id}/delta", h.GetBODelta)
		r.Get("/{id}/scope", h.DiscoverBindingScope)
		r.Get("/{id}/publish-gate", h.ValidatePublishGate)
		r.Get("/{id}/multi-backend", h.GetMultiBackendConfiguration)
		r.Post("/{id}/simulate-impact", h.SimulateLineageImpact)
		r.Get("/{id}/artifacts", h.GenerateBOArtifacts)
		r.Get("/{id}/drift-sentinel", h.DetectSchemaDrift)
		r.Post("/{id}/drift-patch", h.ApplyDriftRepairPatch)
		r.Post("/{id}/lakehouse-compaction", h.RunLakehouseCompaction)
		r.Post("/{id}/ai/explain-delta", h.ExplainDeltaWithAI)

		r.Post("/{id}/ai/detect-anomalies", h.DetectAnomaliesWithAI)
		r.Get("/{id}/workflow", h.GetBOWorkflowStatus)
		r.Post("/{id}/workflow/action", h.ExecuteWorkflowAction)

		r.Get("/{id}/data", h.QueryBORecords)
		r.Post("/{id}/data", h.CreateBORecord)
		r.Put("/{id}/data/{recordId}", h.UpdateBORecord)
		r.Delete("/{id}/data/{recordId}", h.DeleteBORecord)
		r.Put("/{id}", h.UpdateBusinessObject)
		r.Patch("/{id}", h.UpdateBusinessObject)
		r.Delete("/{id}", h.DeleteBusinessObject)
	})
}




// GetBusinessObjectFields returns the list of fields (core + custom) for a BO
func (h *BusinessObjectHandler) GetBusinessObjectFields(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")

	bo, err := h.service.GetBusinessObject(ctx, secCtx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Combine core and custom fields
	fields := make([]map[string]interface{}, 0)
	for _, f := range bo.CoreFields {
		fields = append(fields, map[string]interface{}{
			"id":             f.ID,
			"name":           f.Name,
			"technicalName":  f.TechnicalName,
			"semanticTermId": f.SemanticTermID,
		})
	}
	for _, f := range bo.CustomFields {
		fields = append(fields, map[string]interface{}{
			"id":             f.ID,
			"name":           f.Name,
			"technicalName":  f.TechnicalName,
			"semanticTermId": f.SemanticTermID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fields)
}

func (h *BusinessObjectHandler) ListBusinessObjects(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bos, err := h.service.ListBusinessObjectsComposed(ctx, secCtx)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("ListBusinessObjectsComposed failed, falling back: %v", err)
		bos, err = h.service.ListBusinessObjects(ctx, secCtx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	parentBOs := make([]*models.BusinessObjectDefinition, 0, len(bos))
	for _, bo := range bos {
		isParent := !bo.ActiveSubtypeFilter.Valid ||
			(bo.ActiveSubtypeFilter.Valid && bo.ActiveSubtypeFilter.String == bo.Key)
		if isParent {
			parentBOs = append(parentBOs, bo)
		}
	}

	var filteredBOs []*models.BusinessObjectDefinition
	var summary *security.EntitlementsSummary

	if h.entitlements == nil {
		filteredBOs = parentBOs
		summary = &security.EntitlementsSummary{TotalBO: len(filteredBOs), VisibleBO: len(filteredBOs)}
	} else {
		filtered, entSummary, err := h.entitlements.ListVisibleBOs(ctx, secCtx, parentBOs, nil)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to get BO entitlements, showing all: %v", err)
			filteredBOs = parentBOs
			summary = &security.EntitlementsSummary{TotalBO: len(filteredBOs), VisibleBO: len(filteredBOs)}
		} else {
			summary = entSummary
			filteredBOs = make([]*models.BusinessObjectDefinition, 0, len(filtered))
			for _, fbo := range filtered {
				filteredBOs = append(filteredBOs, fbo.BO)
			}
		}
	}

	format := r.URL.Query().Get("format")
	if format == "array" {
		arr := make([]map[string]interface{}, 0, len(filteredBOs))
		for _, bo := range filteredBOs {
			arr = append(arr, toBusinessObjectResponse(bo))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-BO-Entitlements", security.EntitlementsSummaryHeader(summary))
		json.NewEncoder(w).Encode(arr)
		return
	}

	result := make(map[string]interface{})
	for _, bo := range filteredBOs {
		result[bo.ID] = toBusinessObjectResponse(bo)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-BO-Entitlements", security.EntitlementsSummaryHeader(summary))
	json.NewEncoder(w).Encode(result)
}

func (h *BusinessObjectHandler) CreateBusinessObject(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user ID from auth context
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
		return
	}

	var req models.CreateBusinessObjectRequest
	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	boID := req.ParentID
	if boID == "" {
		boID = "_create"
	}

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, boID)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions to create this business object", http.StatusForbidden)
			return
		}
	}

	// Debug trace to verify scope propagation
	logging.GetLogger().Sugar().Errorf("[HANDLER] CreateBusinessObject scope: tenant=%s ds=%s parent_id=%s name=%s", secCtx.TenantID, secCtx.DatasourceID, req.ParentID, req.Name)

	bo, err := h.service.CreateBusinessObject(ctx, secCtx, req, authInfo.UserID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create BO: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bo)
}

func (h *BusinessObjectHandler) GetBusinessObject(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")

	// IMMEDIATE logging to confirm handler execution
	logging.GetLogger().Sugar().Infof("[HANDLER-ENTRY] GetBusinessObject called: tenant=%s id=%s", secCtx.TenantID, id)

	bo, err := h.service.GetBusinessObject(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("[HANDLER-ERROR] GetBusinessObject service error: tenant=%s id=%s err=%v", secCtx.TenantID, id, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		ents, err := h.entitlements.ForUser(ctx, secCtx)
		if err == nil && ents != nil {
			if _, denied := ents.HiddenBOs[bo.ID]; denied {
				http.Error(w, "Business object not found", http.StatusNotFound)
				return
			}
		}
	}

	logging.GetLogger().Sugar().Infof("DEBUG: API GetBusinessObject called - tenant=%s id=%s boID=%s hasSubtypes=%v", secCtx.TenantID, id, bo.ID, len(bo.Subtypes) > 0)

	// If the service did not populate subtypes (child BO pattern), attempt to list tenant BOs and attach children
	if len(bo.Subtypes) == 0 {
		all, err := h.service.ListBusinessObjects(ctx, secCtx)
		if err == nil {
			attached := 0
			for _, candidate := range all {
				if candidate.ParentID.Valid && candidate.ParentID.String != "" && (candidate.ParentID.String == bo.ID || candidate.ParentID.String == bo.Key) {
					// Map to SubtypeDefinition-like structure
					sd := metadataSubtypeToModel(candidate)
					if bo.Subtypes == nil {
						bo.Subtypes = make(map[string]models.SubtypeDefinition)
					}
					bo.Subtypes[sd.Key] = sd
					attached++
				}
			}
			logging.GetLogger().Sugar().Infof("DEBUG: API GetBusinessObject attached %d child BO(s) to parent %s (tenant %s)", attached, bo.ID, secCtx.TenantID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	// Mark handler for debugging
	w.Header().Set("X-BO-Handler", "api")

	// Return response using toBusinessObjectResponse wrapper for camelCase JSON mapping
	resp := toBusinessObjectResponse(bo)
	json.NewEncoder(w).Encode(resp)
}

// toBusinessObjectResponse converts a BusinessObjectDefinition to a JSON-serializable response
func toBusinessObjectResponse(bo *models.BusinessObjectDefinition) map[string]interface{} {
	driverTableID := ""
	if bo.DriverTableID.Valid {
		driverTableID = bo.DriverTableID.String
	}
	datasourceID := ""
	if bo.DatasourceID.Valid {
		datasourceID = bo.DatasourceID.String
	}
	parentID := ""
	if bo.ParentID.Valid {
		parentID = bo.ParentID.String
	}
	coreID := ""
	if bo.CoreID.Valid {
		coreID = bo.CoreID.String
	}

	logging.GetLogger().Sugar().Infof("[toBusinessObjectResponse] Converting BO %s: driverTableID.Valid=%v driverTableID.String=%s", bo.ID, bo.DriverTableID.Valid, bo.DriverTableID.String)

	return map[string]interface{}{
		"id":                     bo.ID,
		"key":                    bo.Key,
		"name":                   bo.Name,
		"displayName":            bo.DisplayName,
		"technicalName":          bo.TechnicalName,
		"description":            bo.Description,
		"icon":                   bo.Icon,
		"isCore":                 bo.IsCore,
		"coreId":                 coreID, // Workday-style: link to gold copy source BO
		"coreFields":             bo.CoreFields,
		"customFields":           bo.CustomFields,
		"subtypes":               bo.Subtypes,
		"config":                 bo.Config,
		"clonesFrom":             bo.ClonesFrom,
		"cloneParentKey":         bo.CloneParentKey,
		"cloneParentDisplayName": bo.CloneParentDisplayName,
		"category":               bo.Category,
		"parentId":               parentID,
		"instanceCount":          bo.InstanceCount,
		"isActive":               bo.IsActive,
		"createdAt":              bo.CreatedAt,
		"createdBy":              bo.CreatedBy,
		"lastModifiedAt":         bo.LastModifiedAt,
		"lastModifiedBy":         bo.LastModifiedBy,
		"driverTableId":          driverTableID,
		"driverTableName":        bo.DriverTableName,
		"tenantId":               bo.TenantID,
		"datasourceId":           datasourceID,
		"bindings":               bo.Bindings,
	}
}

// helper to convert BusinessObjectDefinition to SubtypeDefinition
func metadataSubtypeToModel(b *models.BusinessObjectDefinition) models.SubtypeDefinition {
	return models.SubtypeDefinition{
		ID:            b.ID,
		Key:           b.Key,
		Name:          b.Name,
		DisplayName:   b.DisplayName,
		TechnicalName: b.TechnicalName,
		Description:   b.Description,
		IsCore:        b.IsCore,
		SubtypeFields: append([]models.FieldDefinition{}, b.CustomFields...),
	}
}

func (h *BusinessObjectHandler) UpdateBusinessObject(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user ID from auth context
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")

	var req models.UpdateBusinessObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logging.GetLogger().Sugar().Errorf("UpdateBusinessObject called; service implementation type: %T", h.service)
	bo, err := h.service.UpdateBusinessObject(ctx, secCtx, id, req, authInfo.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := toBusinessObjectResponse(bo)
	json.NewEncoder(w).Encode(response)
}

func (h *BusinessObjectHandler) DeleteBusinessObject(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user ID from auth context
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")

	if err := h.service.DeleteBusinessObject(ctx, secCtx, id, authInfo.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RenameSubtype renames a subtype within a business object
func (h *BusinessObjectHandler) RenameSubtype(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user ID from auth context
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	subtypeId := chi.URLParam(r, "subtypeId")

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, id)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	var req struct {
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.NewName == "" {
		http.Error(w, "newName is required", http.StatusBadRequest)
		return
	}

	bo, err := h.service.RenameSubtype(ctx, secCtx, id, subtypeId, req.NewName, authInfo.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bo)
}

// DeleteSubtype deletes a subtype from a business object
func (h *BusinessObjectHandler) DeleteSubtype(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get user ID from auth context
	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		http.Error(w, "Unauthorized: missing user ID", http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	subtypeId := chi.URLParam(r, "subtypeId")

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, id)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	bo, err := h.service.DeleteSubtype(ctx, secCtx, id, subtypeId, authInfo.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bo)
}

// GetBusinessObjectRelationships returns related objects and semantic mappings
func (h *BusinessObjectHandler) GetBusinessObjectRelationships(w http.ResponseWriter, r *http.Request) {
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")

	relationships, err := h.service.GetBusinessObjectRelationships(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to get relationships for BO %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(relationships)
}

// CreateBusinessObjectRelationship creates a link between business objects
func (h *BusinessObjectHandler) CreateBusinessObjectRelationship(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sourceBOID := chi.URLParam(r, "id")
	if sourceBOID == "" {
		http.Error(w, "source business object id is required", http.StatusBadRequest)
		return
	}

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, sourceBOID)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	var req struct {
		TargetObjectID   string `json:"targetObjectId"`
		TargetObjectIDSnake string `json:"target_object_id"`
		RelationshipType string `json:"relationshipType"`
		RelationshipTypeSnake string `json:"relationship_type"`
		Cardinality      string `json:"cardinality"`
		Description      string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	targetBOID := req.TargetObjectID
	if targetBOID == "" {
		targetBOID = req.TargetObjectIDSnake
	}
	if targetBOID == "" {
		http.Error(w, "targetObjectId is required", http.StatusBadRequest)
		return
	}

	relType := req.RelationshipType
	if relType == "" {
		relType = req.RelationshipTypeSnake
	}
	if relType == "" {
		relType = "association"
	}

	// 1. Resolve source and target BOs to find driver tables or catalog nodes
	var sourceDriverTable, targetDriverTable sql.NullString
	if h.db != nil {
		_ = h.db.QueryRowContext(ctx, `SELECT driver_table_id FROM public.business_objects WHERE id = $1::uuid`, sourceBOID).Scan(&sourceDriverTable)
		_ = h.db.QueryRowContext(ctx, `SELECT driver_table_id FROM public.business_objects WHERE id = $1::uuid`, targetBOID).Scan(&targetDriverTable)

		sourceNodeID := sourceBOID
		if sourceDriverTable.Valid && sourceDriverTable.String != "" {
			sourceNodeID = sourceDriverTable.String
		}

		targetNodeID := targetBOID
		if targetDriverTable.Valid && targetDriverTable.String != "" {
			targetNodeID = targetDriverTable.String
		}

		// Insert or update catalog_edge
		propsJSON, _ := json.Marshal(map[string]interface{}{
			"source_bo_id": sourceBOID,
			"target_bo_id": targetBOID,
			"cardinality":  req.Cardinality,
			"description":  req.Description,
		})

		edgeQuery := `
			INSERT INTO public.catalog_edge (
				id, tenant_id, source_node_id, target_node_id, relationship_type, edge_type, properties, created_at, updated_at
			) VALUES (
				gen_random_uuid(), $1::uuid, $2::uuid, $3::uuid, $4, $4, $5::jsonb, NOW(), NOW()
			)
		`
		_, err = h.db.ExecContext(ctx, edgeQuery, secCtx.TenantID, sourceNodeID, targetNodeID, relType, string(propsJSON))
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: catalog_edge insertion failed: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"sourceObjectId":   sourceBOID,
		"targetObjectId":   targetBOID,
		"relationshipType": relType,
		"cardinality":      req.Cardinality,
		"description":      req.Description,
	})
}

// UpdateBusinessObjectRelationship updates an existing relationship link
func (h *BusinessObjectHandler) UpdateBusinessObjectRelationship(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	relID := chi.URLParam(r, "relationshipId")
	sourceBOID := chi.URLParam(r, "id")

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, sourceBOID)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	var req struct {
		RelationshipType string `json:"relationshipType"`
		Cardinality      string `json:"cardinality"`
		Description      string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if h.db != nil {
		propsJSON, _ := json.Marshal(map[string]interface{}{
			"source_bo_id": sourceBOID,
			"cardinality":  req.Cardinality,
			"description":  req.Description,
		})

		updateQuery := `
			UPDATE public.catalog_edge
			SET relationship_type = COALESCE(NULLIF($1, ''), relationship_type),
			    edge_type = COALESCE(NULLIF($1, ''), edge_type),
			    properties = properties || $2::jsonb,
			    updated_at = NOW()
			WHERE (id = $3::uuid OR (properties->>'edge_id') = $3)
			  AND tenant_id = $4::uuid
		`
		_, _ = h.db.ExecContext(ctx, updateQuery, req.RelationshipType, string(propsJSON), relID, secCtx.TenantID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"relationshipId": relID,
	})
}

// DeleteBusinessObjectRelationship deletes a relationship link
func (h *BusinessObjectHandler) DeleteBusinessObjectRelationship(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	relID := chi.URLParam(r, "relationshipId")
	sourceBOID := chi.URLParam(r, "id")

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, sourceBOID)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	if h.db != nil {
		delQuery := `
			DELETE FROM public.catalog_edge
			WHERE tenant_id = $1::uuid
			  AND (
				id = $2::uuid 
				OR (properties->>'edge_id') = $2
				OR (
					(properties->>'source_bo_id' = $3 OR source_node_id = $3::uuid) 
					AND (properties->>'target_bo_id' = $2 OR target_node_id = $2::uuid)
				)
			  )
		`
		_, _ = h.db.ExecContext(ctx, delQuery, secCtx.TenantID, relID, sourceBOID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"deleted": relID,
	})
}

// GetBusinessObjectWithBindings returns the full BO view:
// { bo, fields[], calc_fields[], bindings[], related_bos[] }
func (h *BusinessObjectHandler) GetBusinessObjectWithBindings(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")

	// Get the BO itself
	bo, err := h.service.GetBusinessObject(ctx, secCtx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// For STI parent BOs with no CoreFields, populate from semantic terms mapped to the driving table.
	// This provides the "base fields" for the parent BO that users expect to see.
	if len(bo.CoreFields) == 0 && bo.ActiveSubtypeFilter.Valid && bo.DriverTableID.Valid && bo.DriverTableID.String != "" {
		discoverySvc := boPkg.NewBindingDiscoveryService(h.db)
		if discoverySvc != nil {
			tenantUUID, err := uuid.Parse(secCtx.TenantID)
			if err == nil {
				driverTableUUID, err := uuid.Parse(bo.DriverTableID.String)
				if err == nil {
					terms, err := discoverySvc.DiscoverEligibleTermsForBinding(ctx, tenantUUID, driverTableUUID)
					if err == nil && len(terms) > 0 {
						coreFields := make([]models.FieldDefinition, 0, len(terms))
						for _, term := range terms {
							field := models.FieldDefinition{
								ID:             term.TermNodeID.String(),
								Name:           term.TermKey,
								DisplayName:    term.TermName,
								TechnicalName:  term.TermKey,
								Type:           term.TermType,
								IsCore:         true,
								SemanticTermID: term.TermNodeID.String(),
							}
							coreFields = append(coreFields, field)
						}
						bo.CoreFields = coreFields

						// Persist to business_object_fields so the boresolver can find them later
						if h.db != nil {
							persistCoreFieldsToBOF(ctx, h.db, bo.ID, secCtx.TenantID, coreFields)
						}
					}
				}
			}
		}
	}

	// Get relationships (related BOs)
	relationships, err := h.service.GetBusinessObjectRelationships(ctx, secCtx, id)
	if err != nil {
		relationships = nil
	}

	// Get calc fields for this BO from calc_fields table
	var calcFields []map[string]interface{}
	calcQuery := `
		SELECT id, name, sql_expr, data_type, is_measure, realtime
		FROM public.calc_fields
		WHERE tenant_id = $1 AND object_id = $2
		ORDER BY name
	`
	if h.db != nil {
		rows, err := h.db.QueryContext(ctx, calcQuery, secCtx.TenantID, id)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cf struct {
					ID        string `db:"id"`
					Name      string `db:"name"`
					SQLExpr   string `db:"sql_expr"`
					DataType  string `db:"data_type"`
					IsMeasure bool   `db:"is_measure"`
					Realtime  bool   `db:"realtime"`
				}
				if err := rows.Scan(&cf.ID, &cf.Name, &cf.SQLExpr, &cf.DataType, &cf.IsMeasure, &cf.Realtime); err == nil {
					calcFields = append(calcFields, map[string]interface{}{
						"id":         cf.ID,
						"name":       cf.Name,
						"sql_expr":   cf.SQLExpr,
						"data_type":  cf.DataType,
						"is_measure": cf.IsMeasure,
						"realtime":   cf.Realtime,
					})
				}
			}
		}
	}
	if calcFields == nil {
		calcFields = []map[string]interface{}{}
	}

	// Get bindings for this BO from business_object_bindings
	var bindings []map[string]interface{}
	bindingQuery := `
		SELECT binding_id, binding_name, binding_mode, physical_table_name,
		       valid_time_start_col, valid_time_end_col, transaction_time_start_col,
		       transaction_time_end_col, is_primary, COALESCE(config, '{}'::jsonb) as config
		FROM public.business_object_bindings
		WHERE tenant_id = $1 AND bo_id = $2
		ORDER BY is_primary DESC, binding_name
	`
	if h.db != nil {
		bindingRows, err := h.db.QueryContext(ctx, bindingQuery, secCtx.TenantID, id)
		if err == nil {
			defer bindingRows.Close()
			for bindingRows.Next() {
				var b struct {
					BindingID           string  `db:"binding_id"`
					BindingName         string  `db:"binding_name"`
					BindingMode         string  `db:"binding_mode"`
					PhysicalTableName   string  `db:"physical_table_name"`
					ValidTimeStartCol   *string `db:"valid_time_start_col"`
					ValidTimeEndCol     *string `db:"valid_time_end_col"`
					TransactionStartCol *string `db:"transaction_time_start_col"`
					TransactionEndCol   *string `db:"transaction_time_end_col"`
					IsPrimary           bool    `db:"is_primary"`
					Config              []byte  `db:"config"`
				}
				if err := bindingRows.Scan(&b.BindingID, &b.BindingName, &b.BindingMode, &b.PhysicalTableName,
					&b.ValidTimeStartCol, &b.ValidTimeEndCol, &b.TransactionStartCol, &b.TransactionEndCol,
					&b.IsPrimary, &b.Config); err == nil {
					binding := map[string]interface{}{
						"binding_id":         b.BindingID,
						"binding_name":       b.BindingName,
						"binding_mode":       b.BindingMode,
						"physical_table_name": b.PhysicalTableName,
						"is_primary":         b.IsPrimary,
					}
					if b.ValidTimeStartCol != nil {
						binding["valid_time_start_col"] = *b.ValidTimeStartCol
					}
					if b.ValidTimeEndCol != nil {
						binding["valid_time_end_col"] = *b.ValidTimeEndCol
					}
					if b.TransactionStartCol != nil {
						binding["transaction_time_start_col"] = *b.TransactionStartCol
					}
					if b.TransactionEndCol != nil {
						binding["transaction_time_end_col"] = *b.TransactionEndCol
					}
					if b.Config != nil {
						var cfg map[string]interface{}
						if json.Unmarshal(b.Config, &cfg) == nil {
							binding["config"] = cfg
						}
					}
					bindings = append(bindings, binding)
				}
			}
		}
	}
	if bindings == nil {
		bindings = []map[string]interface{}{}
	}

	// Compose related BOs from relationships
	var relatedBOs []map[string]interface{}
	if relationships != nil {
		for _, rel := range relationships.RelatedObjects {
			relatedBOs = append(relatedBOs, map[string]interface{}{
				"bo_name": rel.RelatedObjectName,
				"edge":    rel.RelationshipType,
				"description": rel.Description,
			})
		}
	}
	if relatedBOs == nil {
		relatedBOs = []map[string]interface{}{}
	}

	// Fields
	var fields []map[string]interface{}
	for _, f := range bo.CoreFields {
		fields = append(fields, map[string]interface{}{
			"name":             f.Name,
			"technical_name":  f.TechnicalName,
			"semantic_term_id": f.SemanticTermID,
			"field_type":      f.Type,
		})
	}
	for _, f := range bo.CustomFields {
		fields = append(fields, map[string]interface{}{
			"name":             f.Name,
			"technical_name":  f.TechnicalName,
			"semantic_term_id": f.SemanticTermID,
			"field_type":       f.Type,
		})
	}

	result := map[string]interface{}{
		"bo":          bo,
		"fields":      fields,
		"calc_fields": calcFields,
		"bindings":    bindings,
		"related_bos": relatedBOs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// IntrospectTable handles POST /api/business-objects/introspect-table
func (h *BusinessObjectHandler) IntrospectTable(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req struct {
		TableID   string `json:"tableId"`
		TableName string `json:"tableName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	lookup := req.TableID
	if lookup == "" {
		lookup = req.TableName
	}
	if lookup == "" {
		http.Error(w, "tableId or tableName is required", http.StatusBadRequest)
		return
	}

	result, err := h.service.IntrospectTable(ctx, secCtx, lookup)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to introspect table %s: %v", lookup, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetBODelta handles GET /api/business-objects/{id}/delta
func (h *BusinessObjectHandler) GetBODelta(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	delta, err := h.service.GetBODelta(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to get BO delta for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(delta)
}

// QueryBORecords handles GET /api/business-objects/{id}/data
func (h *BusinessObjectHandler) QueryBORecords(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")

	// Parse query params
	var req models.BORecordQueryRequest
	req.Search = r.URL.Query().Get("search")
	req.SortBy = r.URL.Query().Get("sortBy")
	req.SortDir = r.URL.Query().Get("sortDir")
	req.SubtypeKey = r.URL.Query().Get("subtypeKey")

	var page, limit int
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	req.Page = page
	req.Limit = limit

	if asOf := r.URL.Query().Get("asOfValidTime"); asOf != "" {
		if t, err := time.Parse(time.RFC3339, asOf); err == nil {
			req.AsOfValidTime = &t
		}
	}

	resp, err := h.service.QueryBORecords(ctx, secCtx, id, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to query BO records for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CreateBORecord handles POST /api/business-objects/{id}/data
func (h *BusinessObjectHandler) CreateBORecord(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authInfo, ok := security.AuthInfoFromContext(r.Context())
	userID := "system"
	if ok && authInfo.UserID != "" {
		userID = authInfo.UserID
	}

	id := chi.URLParam(r, "id")

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, id)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	var req models.BOCrudRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid record payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	record, err := h.service.CreateBORecord(ctx, secCtx, id, req, userID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create BO record for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(record)
}

// UpdateBORecord handles PUT /api/business-objects/{id}/data/{recordId}
func (h *BusinessObjectHandler) UpdateBORecord(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authInfo, ok := security.AuthInfoFromContext(r.Context())
	userID := "system"
	if ok && authInfo.UserID != "" {
		userID = authInfo.UserID
	}

	id := chi.URLParam(r, "id")
	recordId := chi.URLParam(r, "recordId")

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, id)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	var req models.BOCrudRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid record payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	record, err := h.service.UpdateBORecord(ctx, secCtx, id, recordId, req, userID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to update BO record %s for %s: %v", recordId, id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// DeleteBORecord handles DELETE /api/business-objects/{id}/data/{recordId}
func (h *BusinessObjectHandler) DeleteBORecord(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authInfo, ok := security.AuthInfoFromContext(r.Context())
	userID := "system"
	if ok && authInfo.UserID != "" {
		userID = authInfo.UserID
	}

	id := chi.URLParam(r, "id")
	recordId := chi.URLParam(r, "recordId")

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canWrite, err := h.entitlements.CanWrite(ctx, secCtx, id)
		if err == nil && !canWrite {
			http.Error(w, "Forbidden: insufficient permissions for this business object", http.StatusForbidden)
			return
		}
	}

	if err := h.service.DeleteBORecord(ctx, secCtx, id, recordId, userID); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to delete BO record %s for %s: %v", recordId, id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SynthesizeBOWithAI handles POST /api/business-objects/ai/synthesize
func (h *BusinessObjectHandler) SynthesizeBOWithAI(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req models.BOAISynthesizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	boID := req.TableID
	if boID == "" {
		boID = req.TableName
	}
	if boID == "" {
		boID = "_synthesize"
	}

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canRun, err := h.entitlements.CanRunAI(ctx, secCtx, boID)
		if err == nil && !canRun {
			http.Error(w, "Forbidden: AI access denied", http.StatusForbidden)
			return
		}
	}

	resp, err := h.service.SynthesizeBOWithAI(ctx, secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to synthesize BO with AI: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// TranslateNLToQueryDef handles POST /api/business-objects/ai/nlq
func (h *BusinessObjectHandler) TranslateNLToQueryDef(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req models.BOAINLQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	boID := req.BOIDOrKey
	if boID == "" {
		boID = "_nlq"
	}

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canRun, err := h.entitlements.CanRunAI(ctx, secCtx, boID)
		if err == nil && !canRun {
			http.Error(w, "Forbidden: AI access denied", http.StatusForbidden)
			return
		}
	}

	resp, err := h.service.TranslateNLToQueryDef(ctx, secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to translate NLQ: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ExplainDeltaWithAI handles POST /api/business-objects/{id}/ai/explain-delta
func (h *BusinessObjectHandler) ExplainDeltaWithAI(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	var req models.BOAIExplainDeltaRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.BOIDOrKey = id

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canRun, err := h.entitlements.CanRunAI(ctx, secCtx, id)
		if err == nil && !canRun {
			http.Error(w, "Forbidden: AI access denied", http.StatusForbidden)
			return
		}
	}

	resp, err := h.service.ExplainDeltaWithAI(ctx, secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to explain delta for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DetectAnomaliesWithAI handles POST /api/business-objects/{id}/ai/detect-anomalies
func (h *BusinessObjectHandler) DetectAnomaliesWithAI(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	var req models.BOAIAnomalyDetectRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	req.BOIDOrKey = id

	if !secCtx.IsGlobalAdmin && h.entitlements != nil {
		canRun, err := h.entitlements.CanRunAI(ctx, secCtx, id)
		if err == nil && !canRun {
			http.Error(w, "Forbidden: AI access denied", http.StatusForbidden)
			return
		}
	}

	resp, err := h.service.DetectAnomaliesWithAI(ctx, secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to detect anomalies for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetBOWorkflowStatus handles GET /api/business-objects/{id}/workflow
func (h *BusinessObjectHandler) GetBOWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := h.service.GetBOWorkflowStatus(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to get workflow status for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ExecuteWorkflowAction handles POST /api/business-objects/{id}/workflow/action
func (h *BusinessObjectHandler) ExecuteWorkflowAction(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authInfo, ok := security.AuthInfoFromContext(r.Context())
	userID := "system"
	if ok && authInfo.UserID != "" {
		userID = authInfo.UserID
	}

	id := chi.URLParam(r, "id")
	var req models.BOWorkflowActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ExecuteWorkflowAction(ctx, secCtx, id, req, userID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to execute workflow action for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DiscoverBindingScope handles GET /api/business-objects/{id}/scope
func (h *BusinessObjectHandler) DiscoverBindingScope(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	drivingNodeID := r.URL.Query().Get("drivingNodeId")

	resp, err := h.service.DiscoverBindingScope(ctx, secCtx, id, drivingNodeID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to discover binding scope for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ValidatePublishGate handles GET /api/business-objects/{id}/publish-gate
func (h *BusinessObjectHandler) ValidatePublishGate(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := h.service.ValidatePublishGate(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to validate publish gate for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetMultiBackendConfiguration handles GET /api/business-objects/{id}/multi-backend
func (h *BusinessObjectHandler) GetMultiBackendConfiguration(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := h.service.GetMultiBackendConfiguration(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to get multi-backend configuration for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PerformGraphRAGContext handles POST /api/business-objects/ai/graph-rag
func (h *BusinessObjectHandler) PerformGraphRAGContext(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req models.GraphRAGContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.PerformGraphRAGContext(ctx, secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to execute GraphRAG context: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SimulateLineageImpact handles POST /api/business-objects/{id}/simulate-impact
func (h *BusinessObjectHandler) SimulateLineageImpact(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	var req models.BOLineageImpactSimulationRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	req.BOIDOrKey = id

	resp, err := h.service.SimulateLineageImpact(ctx, secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to simulate lineage impact for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GenerateBOArtifacts handles GET /api/business-objects/{id}/artifacts
func (h *BusinessObjectHandler) GenerateBOArtifacts(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := h.service.GenerateBOArtifacts(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to generate artifacts for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// EvaluateQueryCost handles POST /api/business-objects/evaluate-cost
func (h *BusinessObjectHandler) EvaluateQueryCost(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req models.BOQueryCostEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.EvaluateQueryCost(ctx, secCtx, req)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to evaluate query cost: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DetectSchemaDrift handles GET /api/business-objects/{id}/drift-sentinel
func (h *BusinessObjectHandler) DetectSchemaDrift(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := h.service.DetectSchemaDrift(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to detect schema drift for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ApplyDriftRepairPatch handles POST /api/business-objects/{id}/drift-patch
func (h *BusinessObjectHandler) ApplyDriftRepairPatch(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authInfo, ok := security.AuthInfoFromContext(r.Context())
	userID := "steward"
	if ok && authInfo.UserID != "" {
		userID = authInfo.UserID
	}

	id := chi.URLParam(r, "id")
	var req models.BODriftRepairPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.BOIDOrKey = id

	resp, err := h.service.ApplyDriftRepairPatch(ctx, secCtx, req, userID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to apply drift patch for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RunLakehouseCompaction handles POST /api/business-objects/{id}/lakehouse-compaction
func (h *BusinessObjectHandler) RunLakehouseCompaction(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	resp, err := h.service.RunLakehouseCompaction(ctx, secCtx, id)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to run lakehouse compaction for %s: %v", id, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// persistCoreFieldsToBOF persists discovered semantic terms as core fields in business_object_fields.
// This ensures the boresolver can find them when building SQL queries.
func persistCoreFieldsToBOF(ctx context.Context, db *sqlx.DB, boID, tenantID string, fields []models.FieldDefinition) {
	if len(fields) == 0 {
		return
	}

	query := `
		INSERT INTO public.business_object_fields
			(id, tenant_id, bo_id, term_node_id, field_name, field_role,
			 aggregation_type, binding_requirement, eligibility_source,
			 subtype_scope, is_exposed, inherits_defaults,
			 display_name, technical_name, data_type,
			 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'DIMENSION',
		        'NONE', 'REQUIRED', 'DIRECT',
		        'ALL', true, true,
		        $6, $7, 'string',
		        NOW(), NOW())
		ON CONFLICT (tenant_id, bo_id, field_name) DO UPDATE
			SET term_node_id   = EXCLUDED.term_node_id,
			    display_name   = EXCLUDED.display_name,
			    technical_name = EXCLUDED.technical_name,
			    inherits_defaults = true,
			    updated_at    = NOW()
	`

	for _, f := range fields {
		var termNodeID *string
		if f.SemanticTermID != "" {
			tnid := f.SemanticTermID
			termNodeID = &tnid
		}
		if termNodeID == nil {
			continue // skip fields without semantic term ID
		}

		displayName := f.DisplayName
		if displayName == "" {
			displayName = f.Name
		}
		techName := f.TechnicalName
		if techName == "" {
			techName = f.Name
		}

		_, err := db.ExecContext(ctx, query,
			uuid.New().String(),
			tenantID,
			boID,
			*termNodeID,
			f.Name,
			displayName,
			techName,
		)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to persist core field %s for BO %s: %v", f.Name, boID, err)
		}
	}
}



