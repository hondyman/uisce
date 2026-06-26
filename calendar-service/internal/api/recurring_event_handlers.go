package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"calendar-service/internal/services"
	"calendar-service/internal/utils"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// RecurringEventHandlers handles HTTP requests for recurring events
type RecurringEventHandlers struct {
	recurringService  services.RecurringEventServiceTenantAware
	conflictService   services.ConflictDetectionServiceTenantAware
	timezoneConverter *utils.TimezoneConverter
}

// NewRecurringEventHandlers creates new recurring event handlers
func NewRecurringEventHandlers(
	recurringService services.RecurringEventServiceTenantAware,
	conflictService services.ConflictDetectionServiceTenantAware,
) *RecurringEventHandlers {
	return &RecurringEventHandlers{
		recurringService:  recurringService,
		conflictService:   conflictService,
		timezoneConverter: utils.NewTimezoneConverter(),
	}
}

// CreateRecurrenceRuleRequest represents a request to create a recurrence rule
type CreateRecurrenceRuleRequest struct {
	ProfileID     string `json:"profile_id" binding:"required"`
	RRule         string `json:"rrule" binding:"required"`
	StartTime     string `json:"start_time" binding:"required"`
	EndTime       string `json:"end_time" binding:"required"`
	TimezoneID    string `json:"timezone_id" binding:"required"`
	MaxOccurrence int    `json:"max_occurrence"`
	Description   string `json:"description"`
}

// CreateRecurrenceRule handles POST /api/v1/recurring-events
func (h *RecurringEventHandlers) CreateRecurrenceRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	var req CreateRecurrenceRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid start_time: %v", err), http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid end_time: %v", err), http.StatusBadRequest)
		return
	}

	rule := &services.RecurrenceRule{
		TenantID:      tenantID,
		ProfileID:     req.ProfileID,
		RRule:         req.RRule,
		StartTime:     startTime,
		EndTime:       endTime,
		TimezoneID:    req.TimezoneID,
		MaxOccurrence: req.MaxOccurrence,
		Description:   req.Description,
	}

	if err := h.recurringService.CreateRecurrenceRule(r.Context(), rule); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create recurrence rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// GetRecurrenceRule handles GET /api/v1/recurring-events/{id}
func (h *RecurringEventHandlers) GetRecurrenceRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing recurring event ID", http.StatusBadRequest)
		return
	}

	rule, err := h.recurringService.GetRecurrenceRule(r.Context(), id, tenantID)
	if err != nil {
		http.Error(w, "Recurring event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// ListRecurrenceRules handles GET /api/v1/recurring-events
func (h *RecurringEventHandlers) ListRecurrenceRules(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	profileID := r.URL.Query().Get("profile_id")
	if profileID == "" {
		http.Error(w, "Missing profile_id query parameter", http.StatusBadRequest)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	rules, total, err := h.recurringService.ListRecurrenceRules(r.Context(), profileID, tenantID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list recurrence rules: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data":   rules,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateRecurrenceRuleRequest represents a request to update a recurrence rule
type UpdateRecurrenceRuleRequest struct {
	RRule         string `json:"rrule"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	TimezoneID    string `json:"timezone_id"`
	MaxOccurrence int    `json:"max_occurrence"`
	Description   string `json:"description"`
}

// UpdateRecurrenceRule handles PUT /api/v1/recurring-events/{id}
func (h *RecurringEventHandlers) UpdateRecurrenceRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing recurring event ID", http.StatusBadRequest)
		return
	}

	// Get existing rule
	rule, err := h.recurringService.GetRecurrenceRule(r.Context(), id, tenantID)
	if err != nil {
		http.Error(w, "Recurring event not found", http.StatusNotFound)
		return
	}

	var req UpdateRecurrenceRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Update fields
	if req.RRule != "" {
		rule.RRule = req.RRule
	}
	if req.StartTime != "" {
		st, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			http.Error(w, "Invalid start_time", http.StatusBadRequest)
			return
		}
		rule.StartTime = st
	}
	if req.EndTime != "" {
		et, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			http.Error(w, "Invalid end_time", http.StatusBadRequest)
			return
		}
		rule.EndTime = et
	}
	if req.TimezoneID != "" {
		rule.TimezoneID = req.TimezoneID
	}
	if req.MaxOccurrence > 0 {
		rule.MaxOccurrence = req.MaxOccurrence
	}
	if req.Description != "" {
		rule.Description = req.Description
	}

	if err := h.recurringService.UpdateRecurrenceRule(r.Context(), rule); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update recurrence rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// DeleteRecurrenceRule handles DELETE /api/v1/recurring-events/{id}
func (h *RecurringEventHandlers) DeleteRecurrenceRule(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing recurring event ID", http.StatusBadRequest)
		return
	}

	if err := h.recurringService.DeleteRecurrenceRule(r.Context(), id, tenantID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete recurrence rule: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateOccurrencesRequest represents a request to generate occurrences
type GenerateOccurrencesRequest struct {
	FromDate string `json:"from_date" binding:"required"`
	ToDate   string `json:"to_date" binding:"required"`
}

// GenerateOccurrences handles POST /api/v1/recurring-events/{id}/occurrences
func (h *RecurringEventHandlers) GenerateOccurrences(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Missing recurring event ID", http.StatusBadRequest)
		return
	}

	var req GenerateOccurrencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	from, err := time.Parse(time.RFC3339, req.FromDate)
	if err != nil {
		http.Error(w, "Invalid from_date", http.StatusBadRequest)
		return
	}

	to, err := time.Parse(time.RFC3339, req.ToDate)
	if err != nil {
		http.Error(w, "Invalid to_date", http.StatusBadRequest)
		return
	}

	occurrences, err := h.recurringService.GenerateOccurrences(r.Context(), id, tenantID, from, to)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate occurrences: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"occurrences": occurrences,
		"count":       len(occurrences),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateExceptionRequest represents a request to create an exception
type CreateExceptionRequest struct {
	ExceptionDate string `json:"exception_date" binding:"required"`
	IsDeleted     bool   `json:"is_deleted"`
	NewStartTime  string `json:"new_start_time"`
	NewEndTime    string `json:"new_end_time"`
}

// CreateException handles POST /api/v1/recurring-events/{id}/exceptions
func (h *RecurringEventHandlers) CreateException(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	recurrenceID := r.PathValue("id")
	if recurrenceID == "" {
		http.Error(w, "Missing recurring event ID", http.StatusBadRequest)
		return
	}

	var req CreateExceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	excDate, err := time.Parse(time.RFC3339, req.ExceptionDate)
	if err != nil {
		http.Error(w, "Invalid exception_date", http.StatusBadRequest)
		return
	}

	exc := &services.RecurrenceException{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		RecurrenceID:  recurrenceID,
		ExceptionDate: excDate,
		IsDeleted:     req.IsDeleted,
	}

	if req.NewStartTime != "" {
		st, err := time.Parse(time.RFC3339, req.NewStartTime)
		if err != nil {
			http.Error(w, "Invalid new_start_time", http.StatusBadRequest)
			return
		}
		exc.NewStartTime = &st
	}

	if req.NewEndTime != "" {
		et, err := time.Parse(time.RFC3339, req.NewEndTime)
		if err != nil {
			http.Error(w, "Invalid new_end_time", http.StatusBadRequest)
			return
		}
		exc.NewEndTime = &et
	}

	if err := h.recurringService.CreateException(r.Context(), exc); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create exception: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(exc)
}

// CreateBlackoutPeriodRequest represents a request to create a blackout period
type CreateBlackoutPeriodRequest struct {
	ProfileID  string `json:"profile_id" binding:"required"`
	StartTime  string `json:"start_time" binding:"required"`
	EndTime    string `json:"end_time" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
	TimezoneID string `json:"timezone_id" binding:"required"`
}

// CreateBlackoutPeriod handles POST /api/v1/blackout-periods
func (h *RecurringEventHandlers) CreateBlackoutPeriod(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	var req CreateBlackoutPeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "Invalid start_time", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Invalid end_time", http.StatusBadRequest)
		return
	}

	period := &services.BlackoutPeriod{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		ProfileID:  req.ProfileID,
		StartTime:  startTime,
		EndTime:    endTime,
		Reason:     req.Reason,
		TimezoneID: req.TimezoneID,
	}

	if err := h.conflictService.CreateBlackoutPeriod(r.Context(), period); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create blackout period: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(period)
}

// CheckConflictsRequest represents a request to check for conflicts
type CheckConflictsRequest struct {
	ProfileID string `json:"profile_id" binding:"required"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// CheckConflicts handles POST /api/v1/conflicts/check
func (h *RecurringEventHandlers) CheckConflicts(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "Missing tenant ID", http.StatusUnauthorized)
		return
	}

	var req CheckConflictsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "Invalid start_time", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		http.Error(w, "Invalid end_time", http.StatusBadRequest)
		return
	}

	event := &services.RecurringEventOccurrence{
		StartTime: startTime,
		EndTime:   endTime,
	}

	conflicts, err := h.conflictService.DetectConflicts(r.Context(), req.ProfileID, tenantID, event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check conflicts: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"conflicts":     conflicts,
		"has_conflicts": len(conflicts) > 0,
		"count":         len(conflicts),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAvailableTimezones handles GET /api/v1/timezones
func (h *RecurringEventHandlers) GetAvailableTimezones(w http.ResponseWriter, r *http.Request) {
	// Return common timezones
	timezones := []string{
		"UTC", "America/New_York", "Europe/London", "Asia/Tokyo",
		"Australia/Sydney", "America/Los_Angeles", "Europe/Paris",
	}

	response := map[string]interface{}{
		"timezones": timezones,
		"count":     len(timezones),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ConvertTimeRequest represents a request to convert time between timezones
type ConvertTimeRequest struct {
	Time   string `json:"time" binding:"required"`
	FromTZ string `json:"from_tz" binding:"required"`
	ToTZ   string `json:"to_tz" binding:"required"`
}

// ConvertTime handles POST /api/v1/timezones/convert
func (h *RecurringEventHandlers) ConvertTime(w http.ResponseWriter, r *http.Request) {
	var req ConvertTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	t, err := time.Parse(time.RFC3339, req.Time)
	if err != nil {
		http.Error(w, "Invalid time format", http.StatusBadRequest)
		return
	}

	converted, err := h.timezoneConverter.ConvertTime(t, req.FromTZ, req.ToTZ)
	if err != nil {
		http.Error(w, fmt.Sprintf("Timezone conversion failed: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"original":  t.Format(time.RFC3339),
		"converted": converted.Format(time.RFC3339),
		"from_tz":   req.FromTZ,
		"to_tz":     req.ToTZ,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
