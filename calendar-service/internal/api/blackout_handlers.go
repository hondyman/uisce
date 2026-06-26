package api

import (
	"encoding/json"
	"net/http"
	"time"

	"calendar-service/internal/middleware"
	"calendar-service/internal/services"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/teambition/rrule-go"
)

type BlackoutHandler struct {
	service      services.BlackoutServiceTenantAwareInterface
	auditService services.AuditService
	logger       *logrus.Entry
}

func NewBlackoutHandler(service services.BlackoutServiceTenantAwareInterface, auditService services.AuditService, logger *logrus.Entry) *BlackoutHandler {
	return &BlackoutHandler{
		service:      service,
		auditService: auditService,
		logger:       logger.WithField("handler", "blackout"),
	}
}

// CreateBlackoutRequest represents both one-time and recurring blackouts
type CreateBlackoutRequest struct {
	TenantID           string     `json:"tenant_id"`
	CalendarID         string     `json:"calendar_id"`
	Name               string     `json:"name"`
	StartTime          time.Time  `json:"start_time"`
	EndTime            time.Time  `json:"end_time"`
	RecurrenceRule     string     `json:"recurrence_rule,omitempty"`
	RecurrenceEnd      *time.Time `json:"recurrence_end,omitempty"`
	RecurrenceTimezone string     `json:"recurrence_timezone,omitempty"`
	Reason             string     `json:"reason"`
	ActorID            string     `json:"actor_id"`
}

// CreateBlackoutResponse response after creation
type CreateBlackoutResponse struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	CalendarID     string     `json:"calendar_id"`
	Name           string     `json:"name"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        time.Time  `json:"end_time"`
	RecurrenceRule string     `json:"recurrence_rule,omitempty"`
	RecurrenceEnd  *time.Time `json:"recurrence_end,omitempty"`
	IsRecurring    bool       `json:"is_recurring"`
	CreatedAt      time.Time  `json:"created_at"`
	CreatedBy      string     `json:"created_by"`
}

// Create creates a blackout (one-time or recurring)
// @Summary Create blackout window
// @Tags blackouts
// @Accept json
// @Produce json
// @Param request body CreateBlackoutRequest true "Blackout data"
// @Success 201 {object} CreateBlackoutResponse
// @Router /api/v1/blackouts [post]
func (h *BlackoutHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	var req CreateBlackoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if req.CalendarID == "" {
		http.Error(w, "calendar_id is required", http.StatusBadRequest)
		return
	}

	if req.EndTime.Before(req.StartTime) {
		http.Error(w, "end_time must be after start_time", http.StatusBadRequest)
		return
	}

	// Validate RRULE if provided
	if req.RecurrenceRule != "" {
		if _, err := rrule.StrToRRule(req.RecurrenceRule); err != nil {
			http.Error(w, "invalid recurrence_rule: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Set timezone default
	if req.RecurrenceTimezone == "" {
		req.RecurrenceTimezone = "UTC"
	}

	// Delegate to service layer (includes tenant verification)
	blackoutID, err := h.service.CreateBlackout(ctx, tenantID, req.CalendarID, userID, req.Name)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create blackout")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build response
	response := CreateBlackoutResponse{
		ID:             blackoutID,
		TenantID:       tenantID,
		CalendarID:     req.CalendarID,
		Name:           req.Name,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		RecurrenceRule: req.RecurrenceRule,
		RecurrenceEnd:  req.RecurrenceEnd,
		IsRecurring:    req.RecurrenceRule != "",
		CreatedAt:      time.Now().UTC(),
		CreatedBy:      userID,
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	h.auditService.RecordCreate(ctx, tenantID, "blackout", response.ID,
		map[string]interface{}{
			"name":                req.Name,
			"calendar_id":         req.CalendarID,
			"start_time":          req.StartTime,
			"end_time":            req.EndTime,
			"recurrence_rule":     req.RecurrenceRule,
			"recurrence_end":      req.RecurrenceEnd,
			"recurrence_timezone": req.RecurrenceTimezone,
			"reason":              req.Reason,
		}, userID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetOccurrencesRequest query params for occurrences
type GetOccurrencesRequest struct {
	Start time.Time
	End   time.Time
}

// GetOccurrences returns all occurrences within a date range (expanded for recurring)
// @Summary Get blackout occurrences
// @Tags blackouts
// @Produce json
// @Param id path string true "Blackout ID"
// @Param start query string true "Range start (ISO8601)"
// @Param end query string true "Range end (ISO8601)"
// @Success 200 {array} availability.Occurrence
// @Router /api/v1/blackouts/{id}/occurrences [get]
func (h *BlackoutHandler) GetOccurrences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	blackoutID := vars["id"]

	rangeStart, err := time.Parse(time.RFC3339, r.URL.Query().Get("start"))
	if err != nil {
		http.Error(w, "invalid start parameter: "+err.Error(), http.StatusBadRequest)
		return
	}

	rangeEnd, err := time.Parse(time.RFC3339, r.URL.Query().Get("end"))
	if err != nil {
		http.Error(w, "invalid end parameter: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Delegate to service layer (includes tenant verification)
	occurrences, err := h.service.GetBlackoutOccurrences(ctx, tenantID, blackoutID, rangeStart, rangeEnd)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":     userID,
			"tenant_id":   tenantID,
			"blackout_id": blackoutID,
		}).Warn("Failed to get blackout occurrences")
		http.Error(w, "Failed to get blackout occurrences", http.StatusInternalServerError)
		return
	}

	// Audit logging
	h.logger.WithFields(logrus.Fields{
		"user_id":     userID,
		"tenant_id":   tenantID,
		"blackout_id": blackoutID,
		"range_start": rangeStart,
		"range_end":   rangeEnd,
		"action":      "get_blackout_occurrences",
	}).Debug("Blackout occurrences retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(occurrences)
}

// DeleteBlackout soft deletes a blackout
// @Summary Delete blackout
// @Tags blackouts
// @Param id path string true "Blackout ID"
// @Success 204
// @Router /api/v1/blackouts/{id} [delete]
func (h *BlackoutHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.ExtractUserIDFromContext(ctx)
	tenantID := middleware.ExtractTenantIDFromContext(ctx)

	vars := mux.Vars(r)
	blackoutID := vars["id"]

	// Delegate to service layer (includes tenant verification)
	err := h.service.DeleteBlackout(ctx, tenantID, blackoutID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"tenant_id":   tenantID,
			"blackout_id": blackoutID,
			"error":       err.Error(),
		}).Error("Failed to delete blackout")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Record audit entry (Phase 6: Audit Service Integration)
	h.auditService.RecordDelete(ctx, tenantID, "blackout", blackoutID,
		map[string]interface{}{}, // old values (would need pre-fetch for full audit trail)
		userID)

	w.WriteHeader(http.StatusNoContent)
}
