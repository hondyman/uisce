package handlers

import (
	"database/sql" // Added this import
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/metadata" // Changed import from services to metadata
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// HTTPHandlers handles HTTP requests
type HTTPHandlers struct {
	db           *sql.DB
	boService    *metadata.BusinessObjectService
	cmdManager   *CommandResponseManager
	validator    *ValidationHandler
	errorHandler *ErrorHandler
}

// NewHTTPHandlers creates a new HTTPHandlers
func NewHTTPHandlers(
	db *sql.DB,
	boService *metadata.BusinessObjectService,
	cmdManager *CommandResponseManager,
	validator *ValidationHandler,
	errorHandler *ErrorHandler,
) *HTTPHandlers {
	return &HTTPHandlers{
		db:           db,
		boService:    boService,
		cmdManager:   cmdManager,
		validator:    validator,
		errorHandler: errorHandler,
	}
}

// ============================================================================
// BUSINESS OBJECT ENDPOINTS
// ============================================================================

// CreateBusinessObject handles POST /api/business-objects
func (h *HTTPHandlers) CreateBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	// Validate required headers
	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	var req models.CreateBusinessObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorHandler.BadRequest(w, "Invalid request body")
		return
	}

	// Extract datasource from header if not in request body
	if req.DatasourceID == "" {
		req.DatasourceID = r.Header.Get("X-Tenant-Datasource-ID")
		if req.DatasourceID == "" {
			req.DatasourceID = r.URL.Query().Get("datasource_id")
		}
	}

	fmt.Printf("[HTTP-HANDLER] CreateBusinessObject: tenant=%s parent_id=%s name=%s datasourceID=%s\n", tenantID, req.ParentID, req.Name, req.DatasourceID)

	// Execute command via manager
	result, err := h.cmdManager.ExecuteCreateBO(r.Context(), tenantID, userID, req)
	if err != nil {
		h.errorHandler.CommandFailed(w, err)
		return
	}

	// Write success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// ListBusinessObjects handles GET /api/business-objects
func (h *HTTPHandlers) ListBusinessObjects(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	// Fetch list from service (read-only, always direct)
	secCtx := &security.Context{TenantID: tenantID, DatasourceID: datasourceID}
	bos, err := h.boService.ListBusinessObjects(r.Context(), secCtx)
	if err != nil {
		h.errorHandler.InternalError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bos)
}

// GetBusinessObject handles GET /api/business-objects/{key}
func (h *HTTPHandlers) GetBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	key := chi.URLParam(r, "key")

	logging.GetLogger().Sugar().Infof("[HTTP-HANDLER] GetBusinessObject: tenantID=%s key=%s", tenantID, key)

	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	secCtx := &security.Context{TenantID: tenantID}
	bo, err := h.boService.GetBusinessObject(r.Context(), secCtx, key)
	if err != nil {
		h.errorHandler.NotFound(w, "BO not found")
		return
	}

	logging.GetLogger().Sugar().Infof("[HTTP-HANDLER] Found BO: id=%s key=%s hasSubtypes=%d", bo.ID, bo.Key, len(bo.Subtypes))

	// If subtypes not populated, attach child BOs from list query
	if len(bo.Subtypes) == 0 {
		if tenantID != "" {
			datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
			secCtx := &security.Context{TenantID: tenantID, DatasourceID: datasourceID}
			all, err := h.boService.ListBusinessObjects(r.Context(), secCtx)
			if err == nil {
				logging.GetLogger().Sugar().Infof("[HTTP-HANDLER] Scanning %d total BOs for children of parent.ID=%s", len(all), bo.ID)
				found := 0
				for _, cand := range all {
					if cand.ParentID.Valid && cand.ParentID.String != "" {
						logging.GetLogger().Sugar().Infof("[HTTP-HANDLER] Candidate: id=%s key=%s parentID=%s (matches=%v)", cand.ID, cand.Key, cand.ParentID.String, cand.ParentID.String == bo.ID)
					}
					if cand.ParentID.Valid && cand.ParentID.String != "" && cand.ParentID.String == bo.ID {
						// Build subtype shape
						sd := models.SubtypeDefinition{
							ID:            cand.ID,
							Key:           cand.Key,
							Name:          cand.Name,
							DisplayName:   cand.DisplayName,
							TechnicalName: cand.TechnicalName,
							Description:   cand.Description,
							IsCore:        cand.IsCore,
							SubtypeFields: append([]models.FieldDefinition{}, cand.CustomFields...),
						}
						if bo.Subtypes == nil {
							bo.Subtypes = make(map[string]models.SubtypeDefinition)
						}
						bo.Subtypes[sd.Key] = sd
						found++
					}
				}
				logging.GetLogger().Sugar().Infof("[HTTP-HANDLER] Attached %d child BO(s) to parent %s", found, bo.ID)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	// Mark handler for debugging
	w.Header().Set("X-BO-Handler", "http_handlers")
	json.NewEncoder(w).Encode(bo)
}

// UpdateBusinessObject handles PUT /api/business-objects/{key}
func (h *HTTPHandlers) UpdateBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	key := chi.URLParam(r, "key")

	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	var req models.UpdateBusinessObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorHandler.BadRequest(w, "Invalid request body")
		return
	}

	// Execute command via manager
	result, err := h.cmdManager.ExecuteUpdateBO(r.Context(), tenantID, userID, key, req)
	if err != nil {
		h.errorHandler.CommandFailed(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeleteBusinessObject handles DELETE /api/business-objects/{key}
func (h *HTTPHandlers) DeleteBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	key := chi.URLParam(r, "key")

	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	// Execute command via manager
	err := h.cmdManager.ExecuteDeleteBO(r.Context(), tenantID, userID, key)
	if err != nil {
		h.errorHandler.CommandFailed(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CloneBusinessObject handles POST /api/business-objects/{key}/clone
func (h *HTTPHandlers) CloneBusinessObject(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	key := chi.URLParam(r, "key")

	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	var req models.CloneBORequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorHandler.BadRequest(w, "Invalid request body")
		return
	}

	req.SourceBOKey = key

	// Execute command via manager
	result, err := h.cmdManager.ExecuteCloneBO(r.Context(), tenantID, userID, req)
	if err != nil {
		h.errorHandler.CommandFailed(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// ============================================================================
// INSTANCE ENDPOINTS
// ============================================================================

// CreateInstance handles POST /api/bo/{boKey}/instances
func (h *HTTPHandlers) CreateInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		h.errorHandler.BadRequest(w, "Missing X-Tenant-ID or X-User-ID headers")
		return
	}

	var instance models.BusinessObjectInstance
	if err := json.NewDecoder(r.Body).Decode(&instance); err != nil {
		h.errorHandler.BadRequest(w, "Invalid request body")
		return
	}

	// Execute command via manager
	result, err := h.cmdManager.ExecuteCreateInstance(r.Context(), tenantID, userID, &instance)
	if err != nil {
		h.errorHandler.CommandFailed(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// ListInstances handles GET /api/bo/{boKey}/instances
func (h *HTTPHandlers) ListInstances(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	boKey := chi.URLParam(r, "boKey")

	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	// Parse pagination
	pageNum := 1
	pageSz := 50

	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		pageNum = p
	}
	if sz, err := strconv.Atoi(r.URL.Query().Get("page_size")); err == nil && sz > 0 {
		pageSz = sz
	}

	offset := (pageNum - 1) * pageSz
	instances, total, err := h.boService.ListInstances(r.Context(), tenantID, boKey, offset, pageSz)
	if err != nil {
		h.errorHandler.InternalError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Page", strconv.Itoa(pageNum))
	w.Header().Set("X-Page-Size", strconv.Itoa(pageSz))
	json.NewEncoder(w).Encode(instances)
}

// GetInstance handles GET /api/bo/{boKey}/instances/{instanceID}
func (h *HTTPHandlers) GetInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	instanceID := chi.URLParam(r, "instanceID")

	if err := h.errorHandler.ValidateHeaders(w, tenantID); err != nil {
		return
	}

	instance, err := h.boService.GetInstance(r.Context(), tenantID, instanceID)
	if err != nil {
		h.errorHandler.NotFound(w, "Instance not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instance)
}

// UpdateInstance handles PUT /api/bo/{boKey}/instances/{instanceID}
func (h *HTTPHandlers) UpdateInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	instanceID := chi.URLParam(r, "instanceID")

	if tenantID == "" || userID == "" {
		h.errorHandler.BadRequest(w, "Missing X-Tenant-ID or X-User-ID headers")
		return
	}

	var req struct {
		CoreFields   map[string]interface{} `json:"coreFields,omitempty"`
		CustomFields map[string]interface{} `json:"customFields,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorHandler.BadRequest(w, "Invalid request body")
		return
	}

	// Execute command via manager
	result, err := h.cmdManager.ExecuteUpdateInstance(r.Context(), tenantID, userID, instanceID, req.CoreFields, req.CustomFields)
	if err != nil {
		h.errorHandler.CommandFailed(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeleteInstance handles DELETE /api/bo/{boKey}/instances/{instanceID}
func (h *HTTPHandlers) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	instanceID := chi.URLParam(r, "instanceID")
	boKey := chi.URLParam(r, "boKey")

	if tenantID == "" || userID == "" {
		h.errorHandler.BadRequest(w, "Missing X-Tenant-ID or X-User-ID headers")
		return
	}

	// Execute command via manager
	err := h.cmdManager.ExecuteDeleteInstance(r.Context(), tenantID, userID, instanceID, boKey)
	if err != nil {
		h.errorHandler.CommandFailed(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

// RegisterRoutes registers all routes with the router
func (h *HTTPHandlers) RegisterRoutes(router *chi.Mux) {
	// Business Objects
	router.Post("/api/business-objects", h.CreateBusinessObject)
	router.Get("/api/business-objects", h.ListBusinessObjects)
	router.Get("/api/business-objects/{key}", h.GetBusinessObject)
	router.Put("/api/business-objects/{key}", h.UpdateBusinessObject)
	router.Delete("/api/business-objects/{key}", h.DeleteBusinessObject)
	router.Post("/api/business-objects/{key}/clone", h.CloneBusinessObject)

	// Instances
	router.Post("/api/bo/{boKey}/instances", h.CreateInstance)
	router.Get("/api/bo/{boKey}/instances", h.ListInstances)
	router.Get("/api/bo/{boKey}/instances/{instanceID}", h.GetInstance)
	router.Put("/api/bo/{boKey}/instances/{instanceID}", h.UpdateInstance)
	router.Delete("/api/bo/{boKey}/instances/{instanceID}", h.DeleteInstance)
}
