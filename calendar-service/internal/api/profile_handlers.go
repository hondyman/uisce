package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// ProfileHandler handles schedule profile CRUD operations
type ProfileHandler struct {
	service      services.ProfileServiceTenantAware
	auditService services.AuditService
	logger       *logrus.Entry
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(svc services.ProfileServiceTenantAware, auditService services.AuditService, logger *logrus.Entry) *ProfileHandler {
	return &ProfileHandler{
		service:      svc,
		auditService: auditService,
		logger:       logger.WithField("handler", "profile"),
	}
}

// CreateProfileRequest defines the request body for creating a profile
type CreateProfileRequest struct {
	ProfileName        string          `json:"profile_name"`
	Description        string          `json:"description,omitempty"`
	Calendars          []string        `json:"calendars"`
	ConflictResolution string          `json:"conflict_resolution"`
	Timezone           string          `json:"timezone"`
	Rules              json.RawMessage `json:"rules,omitempty"`
}

// UpdateProfileRequest defines the request body for updating a profile
type UpdateProfileRequest struct {
	ProfileName        *string          `json:"profile_name,omitempty"`
	Description        *string          `json:"description,omitempty"`
	Calendars          *[]string        `json:"calendars,omitempty"`
	ConflictResolution *string          `json:"conflict_resolution,omitempty"`
	Timezone           *string          `json:"timezone,omitempty"`
	Rules              *json.RawMessage `json:"rules,omitempty"`
	Active             *bool            `json:"active,omitempty"`
}

// Create handles POST /api/v1/profiles
// @Summary Create schedule profile
// @Tags profiles
// @Accept json
// @Produce json
// @Param request body CreateProfileRequest true "Profile data"
// @Success 201 {object} services.ScheduleProfile
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/profiles [post]
func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.ProfileName == "" || len(req.Calendars) == 0 {
		writeJSONError(w, http.StatusBadRequest, "profile_name and calendars are required")
		return
	}

	// Get tenant and actor from headers
	tenantID := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required (X-Hasura-Tenant-Id header)")
		return
	}

	actorID := r.Header.Get("X-Hasura-User-Id")
	if actorID == "" {
		actorID = "system"
	}

	// Create profile via service
	input := services.CreateProfileInput{
		ProfileName:        req.ProfileName,
		Description:        req.Description,
		Calendars:          req.Calendars,
		ConflictResolution: req.ConflictResolution,
		Timezone:           req.Timezone,
		Rules:              req.Rules,
		ActorID:            actorID,
	}

	profile, err := h.service.Create(r.Context(), tenantID, input)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create profile")
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

// Update handles PUT /api/v1/profiles/{id}
// @Summary Update schedule profile (bitemporal)
// @Tags profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID"
// @Param request body UpdateProfileRequest true "Profile updates"
// @Success 200 {object} services.ScheduleProfile
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/profiles/{id} [put]
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get tenant and actor from headers
	tenantID := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}

	actorID := r.Header.Get("X-Hasura-User-Id")
	if actorID == "" {
		actorID = "system"
	}

	// Convert to service input
	input := services.UpdateProfileInput{
		ProfileName:        req.ProfileName,
		Description:        req.Description,
		Calendars:          req.Calendars,
		ConflictResolution: req.ConflictResolution,
		Timezone:           req.Timezone,
		Rules:              req.Rules,
		Active:             req.Active,
		ActorID:            actorID,
	}

	profile, err := h.service.Update(r.Context(), tenantID, profileID, input)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update profile")
		if err.Error() == "profile not found or access denied" {
			writeJSONError(w, http.StatusNotFound, "Profile not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// List handles GET /api/v1/profiles
// @Summary List active profiles for tenant
// @Tags profiles
// @Produce json
// @Param limit query int false "Results per page" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} services.ScheduleProfile
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/profiles [get]
func (h *ProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	profiles, err := h.service.ListActive(r.Context(), tenantID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list profiles")
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if profiles == nil {
		profiles = make([]services.ScheduleProfile, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

// Get handles GET /api/v1/profiles/{id}
// @Summary Get profile by ID
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {object} services.ScheduleProfile
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/profiles/{id} [get]
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	tenantID := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}

	profile, err := h.service.GetByID(r.Context(), tenantID, profileID)
	if err != nil {
		h.logger.WithError(err).Debug("Profile not found")
		writeJSONError(w, http.StatusNotFound, "Profile not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// Delete handles DELETE /api/v1/profiles/{id}
// @Summary Delete schedule profile (soft delete)
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/profiles/{id} [delete]
func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	tenantID := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}

	actorID := r.Header.Get("X-Hasura-User-Id")
	if actorID == "" {
		actorID = "system"
	}

	if err := h.service.Delete(r.Context(), tenantID, profileID, actorID); err != nil {
		h.logger.WithError(err).Error("Failed to delete profile")
		if err.Error() == "profile not found or access denied" {
			writeJSONError(w, http.StatusNotFound, "Profile not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListVersions handles GET /api/v1/profiles/{id}/versions
// @Summary List all versions of a profile (including historical)
// @Tags profiles
// @Produce json
// @Param id path string true "Profile ID"
// @Success 200 {array} services.ScheduleProfile
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/profiles/{id}/versions [get]
func (h *ProfileHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["id"]

	tenantID := r.Header.Get("X-Hasura-Tenant-Id")
	if tenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}

	versions, err := h.service.ListVersions(r.Context(), tenantID, profileID)
	if err != nil {
		h.logger.WithError(err).Debug("No versions found")
		writeJSONError(w, http.StatusNotFound, "No versions found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error  string `json:"error"`
	Status int    `json:"status"`
}

// writeJSONError writes an error response in JSON format
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:  message,
		Status: statusCode,
	})
}
