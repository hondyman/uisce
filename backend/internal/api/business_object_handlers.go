package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
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
}

func NewBusinessObjectHandler(service BOService, datasourceResolver security.DatasourceResolver, sqlxDB *sqlx.DB) *BusinessObjectHandler {
	return &BusinessObjectHandler{
		service:            service,
		datasourceResolver: datasourceResolver,
		db:                 sqlxDB,
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
	// Build security context with datasource + region validation
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", handlers.SecurityContextDeps{
		Resolver: h.datasourceResolver,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use Workday-style composed listing (Core + Custom merged)
	bos, err := h.service.ListBusinessObjectsComposed(ctx, secCtx)
	if err != nil {
		// Fallback to regular listing if composition fails
		logging.GetLogger().Sugar().Warnf("ListBusinessObjectsComposed failed, falling back: %v", err)
		bos, err = h.service.ListBusinessObjects(ctx, secCtx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Exclude subtypes from list view (only top-level business objects)
	filtered := make([]*models.BusinessObjectDefinition, 0, len(bos))
	for _, bo := range bos {
		if !bo.ParentID.Valid || bo.ParentID.String == "" {
			filtered = append(filtered, bo)
		}
	}

	// Map to map[string]BO for frontend if needed, or return array.
	// Frontend expects: objects: Record<string, BusinessObject>
	// or array?
	// Based on BusinessObjectsPage.tsx:
	// const objectsArray = Object.entries(data).map... implies object/map response?
	// Wait, if API returns array, Object.entries(array) gives indices.
	// Let's check what frontend expects.
	// Frontend: const { data: objects, ... } = useQuery('/api/business-objects')
	// If objects is array, Object.entries works but keys are '0', '1'.
	// If the frontend code has: const objectsArray = Object.entries(data || {}).map(([id, obj]: [string, any]) => ...
	// It suggests data is a map like { "id1": obj1, "id2": obj2 }.

	result := make(map[string]interface{})
	for _, bo := range filtered {
		result[bo.ID] = toBusinessObjectResponse(bo)
	}

	w.Header().Set("Content-Type", "application/json")
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





