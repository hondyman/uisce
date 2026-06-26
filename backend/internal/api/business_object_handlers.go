package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
"github.com/hondyman/semlayer/backend/internal/handlers"
	"github.com/hondyman/semlayer/backend/internal/logging"
	catalogmeta "github.com/hondyman/semlayer/backend/internal/metadata"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
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
}

type BusinessObjectHandler struct {
	service            BOService
	datasourceResolver security.DatasourceResolver
}

func NewBusinessObjectHandler(service BOService, datasourceResolver security.DatasourceResolver) *BusinessObjectHandler {
	return &BusinessObjectHandler{
		service:            service,
		datasourceResolver: datasourceResolver,
	}
}

func (h *BusinessObjectHandler) RegisterRoutes(r chi.Router) {
	r.Route("/business-objects", func(r chi.Router) {
		r.Get("/", h.ListBusinessObjects)
		r.Post("/", h.CreateBusinessObject)
		// Subtype management routes
		r.Route("/{id}/subtypes", func(r chi.Router) {
			r.Post("/{subtypeId}/rename", h.RenameSubtype)
			r.Delete("/{subtypeId}", h.DeleteSubtype)
		})

		r.Get("/{id}", h.GetBusinessObject)
		r.Get("/{id}/fields", h.GetBusinessObjectFields)
		r.Get("/{id}/relationships", h.GetBusinessObjectRelationships)
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

	// Return raw BO; custom marshal handles nullable fields
	json.NewEncoder(w).Encode(bo)
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
