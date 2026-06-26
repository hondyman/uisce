package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type CalendarHandler struct {
	service      services.CalendarServiceTenantAware
	auditService services.AuditService
	mdmAdapter   *services.MDMAdapter
	logger       *logrus.Entry
}

func NewCalendarHandler(service services.CalendarServiceTenantAware, auditService services.AuditService, logger *logrus.Entry) *CalendarHandler {
	return &CalendarHandler{
		service:      service,
		auditService: auditService,
		mdmAdapter:   nil,
		logger:       logger.WithField("handler", "calendar"),
	}
}

// SetMDMAdapter sets the MDM adapter for calendar operations
func (h *CalendarHandler) SetMDMAdapter(adapter *services.MDMAdapter) {
	h.mdmAdapter = adapter
}

// CreateCalendarRequest represents a new calendar
type CreateCalendarRequest struct {
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Timezone    string                 `json:"timezone"` // e.g., "America/New_York"
	Type        string                 `json:"type"`     // e.g., "fulfillment", "support", "custom"
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ActorID     string                 `json:"actor_id"`
}

// CreateCalendarResponse response after creation
type CreateCalendarResponse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Timezone    string    `json:"timezone"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
}

// Create creates a new calendar
// @Summary Create calendar
// @Tags calendars
// @Accept json
// @Produce json
// @Param request body CreateCalendarRequest true "Calendar data"
// @Success 201 {object} CreateCalendarResponse
// @Router /api/v1/calendars [post]
func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	var req CreateCalendarRequest
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

	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	// Delegate to service layer (includes validation, tenant verification, audit logging)
	calendar, err := h.service.Create(ctx, tenantID, userID, req.Name, req.Description, req.Timezone)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	h.auditService.RecordCreate(ctx, tenantID, "calendar", calendar.ID,
		map[string]interface{}{
			"name":        calendar.Name,
			"description": calendar.Description,
			"timezone":    calendar.Region,
			"type":        "calendar",
		}, userID)

	// Format response
	response := CreateCalendarResponse{
		ID:          calendar.ID,
		TenantID:    calendar.TenantID,
		Name:        calendar.Name,
		Description: calendar.Description,
		Timezone:    calendar.Region,
		Type:        "calendar",
		CreatedAt:   calendar.CreatedAt,
		CreatedBy:   calendar.CreatedBy,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetCalendarResponse represents a calendar
type GetCalendarResponse struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Timezone    string    `json:"timezone"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Get retrieves a calendar by ID
// @Summary Get calendar
// @Tags calendars
// @Produce json
// @Param id path string true "Calendar ID"
// @Success 200 {object} GetCalendarResponse
// @Router /api/v1/calendars/{id} [get]
func (h *CalendarHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	calendarID := vars["id"]

	// Delegate to service layer (includes tenant verification)
	calendar, err := h.service.GetByID(ctx, tenantID, calendarID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Audit logging
	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
		"action":      "get_calendar",
	}).Debug("Calendar retrieved")

	// Format response
	response := GetCalendarResponse{
		ID:          calendar.ID,
		TenantID:    calendar.TenantID,
		Name:        calendar.Name,
		Description: calendar.Description,
		Timezone:    calendar.Region,
		Type:        "calendar",
		CreatedAt:   calendar.CreatedAt,
		CreatedBy:   calendar.CreatedBy,
		UpdatedAt:   calendar.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateCalendarRequest represents an update to a calendar
type UpdateCalendarRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Timezone    string                 `json:"timezone,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ActorID     string                 `json:"actor_id"`
}

// Update updates a calendar
// @Summary Update calendar
// @Tags calendars
// @Accept json
// @Produce json
// @Param id path string true "Calendar ID"
// @Param request body UpdateCalendarRequest true "Update data"
// @Success 200 {object} GetCalendarResponse
// @Router /api/v1/calendars/{id} [put]
func (h *CalendarHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	calendarID := vars["id"]

	var req UpdateCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Timezone != "" {
		updates["timezone"] = req.Timezone
	}

	// Delegate to service layer (includes tenant verification, validation, audit logging)
	calendar, err := h.service.Update(ctx, tenantID, calendarID, userID, updates)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	h.auditService.RecordUpdate(ctx, tenantID, "calendar", calendarID,
		map[string]interface{}{}, // old values (would need pre-fetch for full diff)
		updates,                  // new values
		userID)

	// Format response
	response := GetCalendarResponse{
		ID:          calendar.ID,
		TenantID:    calendar.TenantID,
		Name:        calendar.Name,
		Description: calendar.Description,
		Timezone:    calendar.Region,
		Type:        "calendar",
		CreatedAt:   calendar.CreatedAt,
		CreatedBy:   calendar.CreatedBy,
		UpdatedAt:   calendar.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Delete soft deletes a calendar
// @Summary Delete calendar
// @Tags calendars
// @Param id path string true "Calendar ID"
// @Success 204
// @Router /api/v1/calendars/{id} [delete]
func (h *CalendarHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	calendarID := vars["id"]

	// Delegate to service layer (includes tenant verification, audit logging)
	err := h.service.Delete(ctx, tenantID, calendarID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	h.auditService.RecordDelete(ctx, tenantID, "calendar", calendarID,
		map[string]interface{}{}, // old values (would need pre-fetch for full audit trail)
		userID)

	w.WriteHeader(http.StatusNoContent)
}

// ListCalendarsResponse represents a list of calendars
type ListCalendarsResponse struct {
	Calendars []GetCalendarResponse `json:"calendars"`
	Total     int                   `json:"total"`
}

// List retrieves all calendars for a tenant
// @Summary List calendars
// @Tags calendars
// @Produce json
// @Success 200 {object} ListCalendarsResponse
// @Router /api/v1/calendars [get]
func (h *CalendarHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	// Parse pagination: ?limit=10&offset=0
	limit := 10
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := parseInt(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := parseInt(o); err == nil && val >= 0 {
			offset = val
		}
	}

	// Delegate to service layer (service returns only tenant's calendars)
	calendars, err := h.service.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Audit logging
	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"tenant_id": tenantID,
		"action":    "list_calendars",
		"count":     len(calendars),
	}).Debug("Calendars listed")

	// Format response
	calendarResponses := make([]GetCalendarResponse, len(calendars))
	for i, cal := range calendars {
		calendarResponses[i] = GetCalendarResponse{
			ID:          cal.ID,
			TenantID:    cal.TenantID,
			Name:        cal.Name,
			Description: cal.Description,
			Timezone:    cal.Region,
			Type:        "calendar",
			CreatedAt:   cal.CreatedAt,
			CreatedBy:   cal.CreatedBy,
			UpdatedAt:   cal.UpdatedAt,
		}
	}

	response := ListCalendarsResponse{
		Calendars: calendarResponses,
		Total:     len(calendarResponses),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// Helper Methods
// ============================================================================

// handleServiceError converts service layer errors to HTTP responses
func (h *CalendarHandler) handleServiceError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	// Map service errors to HTTP status codes
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Generic not found (doesn't leak cross-tenant info)
		http.Error(w, "Resource not found", http.StatusNotFound)

	case errors.Is(err, context.DeadlineExceeded):
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)

	case errors.Is(err, context.Canceled):
		http.Error(w, "Request canceled", http.StatusBadRequest)

	default:
		// Check for specific service layer errors
		errMsg := err.Error()
		if errMsg == "access denied" || errMsg == "tenant access denied" {
			http.Error(w, "Access denied", http.StatusForbidden)
		} else if errMsg == "not found" || errMsg == "calendar not found" {
			http.Error(w, "Resource not found", http.StatusNotFound)
		} else {
			h.logger.WithError(err).Error("Unhandled error in calendar handler")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// parseInt safely parses string to integer
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
