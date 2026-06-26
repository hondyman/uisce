package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type TenantHandler struct {
	service      services.TenantServiceInterface
	auditService services.AuditService
	logger       *logrus.Entry
}

func NewTenantHandler(service services.TenantServiceInterface, auditService services.AuditService, logger *logrus.Entry) *TenantHandler {
	return &TenantHandler{
		service:      service,
		auditService: auditService,
		logger:       logger.WithField("handler", "tenant"),
	}
}

// CreateTenantRequest represents a new tenant
type CreateTenantRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Email       string                 `json:"email"`
	Phone       string                 `json:"phone"`
	Country     string                 `json:"country"`
	Timezone    string                 `json:"timezone"` // Default timezone for the tenant
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ActorID     string                 `json:"actor_id"`
}

// CreateTenantResponse response after creation
type CreateTenantResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "active", "suspended", "pending"
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	APIKey    string    `json:"api_key,omitempty"` // Sensitive, only returned once
}

// Create creates a new tenant
// @Summary Create tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param request body CreateTenantRequest true "Tenant data"
// @Success 201 {object} CreateTenantResponse
// @Router /api/v1/tenants [post]
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	// For tenant creation, verify user has admin role
	roles := middleware.ExtractRolesFromContext(ctx)
	hasAdminRole := middleware.HasRole(ctx, "admin")

	if !hasAdminRole {
		h.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"roles":   roles,
			"action":  "create_tenant",
		}).Warn("Unauthorized: requires admin role")
		http.Error(w, "Insufficient permissions to create tenant", http.StatusForbidden)
		return
	}

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Delegate to service layer (includes admin verification)
	tenantID, err := h.service.CreateTenant(ctx, userID, req.Name, req.Description)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create tenant")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := CreateTenantResponse{
		ID:        tenantID,
		Name:      req.Name,
		Status:    "active",
		CreatedAt: time.Now().UTC(),
		CreatedBy: userID,
		APIKey:    "sk_live_" + time.Now().Format("20060102150405"), // placeholder
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	// Use tenantID as both the entity and tenant context
	h.auditService.RecordCreate(ctx, tenantID, "tenant", tenantID,
		map[string]interface{}{
			"name":        req.Name,
			"description": req.Description,
			"email":       req.Email,
			"phone":       req.Phone,
			"country":     req.Country,
			"timezone":    req.Timezone,
			"status":      "active",
		}, userID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetTenantResponse represents a tenant
type GetTenantResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Country     string    `json:"country"`
	Timezone    string    `json:"timezone"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Get retrieves a tenant by ID
// @Summary Get tenant
// @Tags tenants
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} GetTenantResponse
// @Router /api/v1/tenants/{id} [get]
func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	requestedTenantID := vars["id"]

	// Verify cross-tenant access control
	if requestedTenantID != tenantID {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"tenant_id":  tenantID,
			"request_id": requestedTenantID,
			"action":     "get_tenant",
		}).Warn("Cross-tenant access attempted")
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Delegate to service layer
	tenant, err := h.service.GetTenant(ctx, tenantID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to get tenant")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := GetTenantResponse{
		ID:          tenantID,
		Name:        fmt.Sprintf("%v", tenant["name"]),
		Description: fmt.Sprintf("%v", tenant["description"]),
		Email:       fmt.Sprintf("%v", tenant["email"]),
		Phone:       fmt.Sprintf("%v", tenant["phone"]),
		Country:     fmt.Sprintf("%v", tenant["country"]),
		Timezone:    fmt.Sprintf("%v", tenant["timezone"]),
		Status:      fmt.Sprintf("%v", tenant["status"]),
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   userID,
		UpdatedAt:   time.Now().UTC(),
	}

	// Audit logging
	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"tenant_id": tenantID,
		"action":    "get_tenant",
	}).Debug("Tenant retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateTenantRequest represents an update to a tenant
type UpdateTenantRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Country     string `json:"country,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	ActorID     string `json:"actor_id"`
}

// Update updates a tenant
// @Summary Update tenant
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param request body UpdateTenantRequest true "Update data"
// @Success 200 {object} GetTenantResponse
// @Router /api/v1/tenants/{id} [put]
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	requestedTenantID := vars["id"]

	// Verify cross-tenant access control
	if requestedTenantID != tenantID {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"tenant_id":  tenantID,
			"request_id": requestedTenantID,
			"action":     "update_tenant",
		}).Warn("Cross-tenant access attempted")
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Delegate to service layer
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"email":       req.Email,
		"phone":       req.Phone,
		"country":     req.Country,
		"timezone":    req.Timezone,
	}

	err := h.service.UpdateTenant(ctx, tenantID, userID, updates)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to update tenant")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Retrieve updated tenant
	updated, err := h.service.GetTenant(ctx, tenantID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to retrieve updated tenant")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := GetTenantResponse{
		ID:          tenantID,
		Name:        fmt.Sprintf("%v", updated["name"]),
		Description: fmt.Sprintf("%v", updated["description"]),
		Email:       fmt.Sprintf("%v", updated["email"]),
		Phone:       fmt.Sprintf("%v", updated["phone"]),
		Country:     fmt.Sprintf("%v", updated["country"]),
		Timezone:    fmt.Sprintf("%v", updated["timezone"]),
		Status:      fmt.Sprintf("%v", updated["status"]),
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   userID,
		UpdatedAt:   time.Now().UTC(),
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	h.auditService.RecordUpdate(ctx, tenantID, "tenant", tenantID,
		map[string]interface{}{}, // old values (would need pre-fetch for full diff)
		updates,                  // new values
		userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TenantConfig represents tenant-specific configuration
type TenantConfig struct {
	ID                     string                 `json:"id"`
	TenantID               string                 `json:"tenant_id"`
	DefaultTimezone        string                 `json:"default_timezone"`
	DefaultAvailability    string                 `json:"default_availability"`
	LocalizationPreference string                 `json:"localization_preference"`
	CustomSettings         map[string]interface{} `json:"custom_settings,omitempty"`
	UpdatedAt              time.Time              `json:"updated_at"`
}

// GetConfig retrieves tenant configuration
// @Summary Get tenant config
// @Tags tenants
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} TenantConfig
// @Router /api/v1/tenants/{id}/config [get]
func (h *TenantHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	requestedTenantID := vars["id"]

	// Verify cross-tenant access control
	if requestedTenantID != tenantID {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"tenant_id":  tenantID,
			"request_id": requestedTenantID,
			"action":     "get_tenant_config",
		}).Warn("Cross-tenant access attempted")
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Delegate to service layer
	config, err := h.service.GetTenantConfig(ctx, tenantID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to get tenant config")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := TenantConfig{
		ID:                     fmt.Sprintf("config-%s", tenantID),
		TenantID:               tenantID,
		DefaultTimezone:        fmt.Sprintf("%v", config["default_timezone"]),
		DefaultAvailability:    fmt.Sprintf("%v", config["default_availability"]),
		LocalizationPreference: fmt.Sprintf("%v", config["localization_preference"]),
		CustomSettings:         config,
		UpdatedAt:              time.Now().UTC(),
	}

	// Audit logging
	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"tenant_id": tenantID,
		"action":    "get_tenant_config",
	}).Debug("Tenant config retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateConfigRequest represents an update to tenant configuration
type UpdateConfigRequest struct {
	DefaultTimezone        string                 `json:"default_timezone,omitempty"`
	DefaultAvailability    string                 `json:"default_availability,omitempty"`
	LocalizationPreference string                 `json:"localization_preference,omitempty"`
	CustomSettings         map[string]interface{} `json:"custom_settings,omitempty"`
	ActorID                string                 `json:"actor_id"`
}

// UpdateConfig updates tenant configuration
// @Summary Update tenant config
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param request body UpdateConfigRequest true "Config data"
// @Success 200 {object} TenantConfig
// @Router /api/v1/tenants/{id}/config [put]
func (h *TenantHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	requestedTenantID := vars["id"]

	// Verify cross-tenant access control
	if requestedTenantID != tenantID {
		h.logger.WithFields(logrus.Fields{
			"user_id":    userID,
			"tenant_id":  tenantID,
			"request_id": requestedTenantID,
			"action":     "update_tenant_config",
		}).Warn("Cross-tenant access attempted")
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Delegate to service layer
	configUpdates := map[string]interface{}{
		"default_timezone":        req.DefaultTimezone,
		"default_availability":    req.DefaultAvailability,
		"localization_preference": req.LocalizationPreference,
		"custom_settings":         req.CustomSettings,
	}

	err := h.service.UpdateTenantConfig(ctx, tenantID, userID, configUpdates)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to update tenant config")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Retrieve updated config
	config, err := h.service.GetTenantConfig(ctx, tenantID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"tenant_id": tenantID,
			"error":     err.Error(),
		}).Error("Failed to retrieve updated config")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := TenantConfig{
		ID:                     fmt.Sprintf("config-%s", tenantID),
		TenantID:               tenantID,
		DefaultTimezone:        fmt.Sprintf("%v", config["default_timezone"]),
		DefaultAvailability:    fmt.Sprintf("%v", config["default_availability"]),
		LocalizationPreference: fmt.Sprintf("%v", config["localization_preference"]),
		CustomSettings:         config,
		UpdatedAt:              time.Now().UTC(),
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	h.auditService.RecordUpdate(ctx, tenantID, "tenant_config", fmt.Sprintf("config-%s", tenantID),
		map[string]interface{}{}, // old values (would need pre-fetch for full diff)
		configUpdates,            // new values
		userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
