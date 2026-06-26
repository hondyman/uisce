package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SchedulerHandlers provides scheduler API endpoints
type SchedulerHandlers struct {
	schedulerService services.SchedulerService
}

// NewSchedulerHandlers creates new scheduler handlers
func NewSchedulerHandlers(ss services.SchedulerService) *SchedulerHandlers {
	return &SchedulerHandlers{
		schedulerService: ss,
	}
}

// CreateScheduledJob creates a new scheduled job (POST /api/v1/schedules)
func (h *SchedulerHandlers) CreateScheduledJob(w http.ResponseWriter, r *http.Request) {
	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	var job services.ScheduledJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Validate required fields
	if job.OperationType == "" || job.ScheduleType == "" {
		sendError(w, http.StatusBadRequest, "Missing required fields: operation_type, schedule_type")
		return
	}

	if job.ScheduleType == "cron" && job.CronExpression == "" {
		sendError(w, http.StatusBadRequest, "Cron expression required for cron schedule type")
		return
	}

	scheduleID, err := h.schedulerService.CreateSchedule(ctx, &job)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to create schedule", err.Error())
		return
	}

	response := map[string]interface{}{
		"id":          scheduleID,
		"message":     "Schedule created successfully",
		"next_run_at": job.StartTime,
	}

	sendJSON(w, http.StatusCreated, response)
}

// GetSchedule retrieves a scheduled job (GET /api/v1/schedules/:scheduleId)
func (h *SchedulerHandlers) GetSchedule(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "scheduleId")
	scheduleID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	job, err := h.schedulerService.GetSchedule(ctx, scheduleID)
	if err != nil {
		if err.Error() == "schedule not found" {
			sendError(w, http.StatusNotFound, "Schedule not found")
		} else {
			SendErrorResponse(w, 500, "Failed to get schedule", err.Error())
		}
		return
	}

	sendJSON(w, http.StatusOK, job)
}

// ListSchedules lists all schedules for a tenant (GET /api/v1/schedules)
func (h *SchedulerHandlers) ListSchedules(w http.ResponseWriter, r *http.Request) {
	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	tenantID, err := uuid.Parse(tenantNorm)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	ctx := setupAuthContext(r.Context(), tenantNorm)

	schedules, err := h.schedulerService.ListSchedules(ctx, tenantID)
	if err != nil {
		SendErrorResponse(w, 500, "Failed to list schedules", err.Error())
		return
	}

	response := map[string]interface{}{
		"schedules": schedules,
		"total":     len(schedules),
	}

	sendJSON(w, http.StatusOK, response)
}

// PauseSchedule pauses a scheduled job (POST /api/v1/schedules/:scheduleId/pause)
func (h *SchedulerHandlers) PauseSchedule(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "scheduleId")
	scheduleID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	err = h.schedulerService.PauseSchedule(ctx, scheduleID)
	if err != nil {
		if err.Error() == "schedule not found" {
			sendError(w, http.StatusNotFound, "Schedule not found")
		} else {
			SendErrorResponse(w, 500, "Failed to pause schedule", err.Error())
		}
		return
	}

	response := map[string]string{
		"message": "Schedule paused successfully",
		"id":      scheduleID.String(),
	}

	sendJSON(w, http.StatusOK, response)
}

// ResumeSchedule resumes a paused schedule (POST /api/v1/schedules/:scheduleId/resume)
func (h *SchedulerHandlers) ResumeSchedule(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "scheduleId")
	scheduleID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	err = h.schedulerService.ResumeSchedule(ctx, scheduleID)
	if err != nil {
		if err.Error() == "schedule not found" {
			sendError(w, http.StatusNotFound, "Schedule not found")
		} else {
			SendErrorResponse(w, 500, "Failed to resume schedule", err.Error())
		}
		return
	}

	response := map[string]string{
		"message": "Schedule resumed successfully",
		"id":      scheduleID.String(),
	}

	sendJSON(w, http.StatusOK, response)
}

// DeleteSchedule deletes a schedule (DELETE /api/v1/schedules/:scheduleId)
func (h *SchedulerHandlers) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := chi.URLParam(r, "scheduleId")
	scheduleID, err := uuid.Parse(vars)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid schedule ID")
		return
	}

	tenantNorm := normalizeTenantID(jwtmiddleware.GetClaimsFromContext(r).TenantID)
	ctx := setupAuthContext(r.Context(), tenantNorm)

	err = h.schedulerService.DeleteSchedule(ctx, scheduleID)
	if err != nil {
		if err.Error() == "schedule not found" {
			sendError(w, http.StatusNotFound, "Schedule not found")
		} else {
			SendErrorResponse(w, 500, "Failed to delete schedule", err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
