package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/reporting"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type ReportScheduleHandler struct {
	db *sqlx.DB
}

func NewReportScheduleHandler(db *sqlx.DB) *ReportScheduleHandler {
	return &ReportScheduleHandler{db: db}
}

// resolveTenantID resolves the tenant from verified auth context only.
func (h *ReportScheduleHandler) resolveTenantID(r *http.Request) (uuid.UUID, error) {
	// 1. Try security.AuthInfoFromContext
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok {
		if active, ok := auth.ActiveTenant(); ok {
			if tid, err := uuid.Parse(active); err == nil && tid != uuid.Nil {
				return tid, nil
			}
		}
	}

	// 2. Try identity context
	if tidStr, ok := identity.TenantIDFromContext(r.Context()); ok {
		if tid, err := uuid.Parse(tidStr); err == nil && tid != uuid.Nil {
			return tid, nil
		}
	}

	// 3. Try jwtmiddleware claims
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		if tid, err := uuid.Parse(claims.TenantID); err == nil && tid != uuid.Nil {
			return tid, nil
		}
	}

	return uuid.Nil, errors.New("unauthorized: missing or invalid tenant identification")
}

func (h *ReportScheduleHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/reports", func(r chi.Router) {
		r.Get("/calendars", h.ListCalendars)
		r.Get("/schedules", h.ListSchedules)
		r.Post("/schedules", h.CreateSchedule)
		r.Get("/schedules/{id}", h.GetSchedule)
		r.Post("/schedules/{id}/run", h.TriggerScheduleRun)
		r.Get("/schedules/{id}/batches", h.ListScheduleBatches)
		r.Get("/batches/{id}/telemetry", h.GetBatchTelemetry)
		r.Post("/batches/{id}/retry-dlq", h.RetryBatchDLQ)
		r.Post("/calendars/{id}/sync", h.SyncCalendarHolidays)
	})
}

func (h *ReportScheduleHandler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.resolveTenantID(r)
	if err != nil {
		// Fallback mock calendars if tenant id header not present or invalid
		calendars := []map[string]interface{}{
			{"calendar_code": "NYSE", "calendar_name": "New York Stock Exchange", "timezone": "America/New_York"},
			{"calendar_code": "LSE", "calendar_name": "London Stock Exchange", "timezone": "Europe/London"},
			{"calendar_code": "TARGET2", "calendar_name": "Trans-European Automated Real-time Gross Settlement", "timezone": "Europe/Frankfurt"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendars)
		return
	}

	var calendars []struct {
		ID           uuid.UUID `json:"id" db:"id"`
		CalendarCode string    `json:"calendar_code" db:"calendar_code"`
		CalendarName string    `json:"calendar_name" db:"calendar_name"`
		Timezone     string    `json:"timezone" db:"timezone"`
		IsActive     bool      `json:"is_active" db:"is_active"`
	}

	err = h.db.SelectContext(r.Context(), &calendars, `
		SELECT id, calendar_code, calendar_name, timezone, is_active 
		FROM public.tenant_exchange_calendars 
		WHERE tenant_id = $1 AND is_active = true
	`, tenantID)

	if err != nil || len(calendars) == 0 {
		defaultCalendars := []map[string]interface{}{
			{"calendar_code": "NYSE", "calendar_name": "New York Stock Exchange", "timezone": "America/New_York"},
			{"calendar_code": "LSE", "calendar_name": "London Stock Exchange", "timezone": "Europe/London"},
			{"calendar_code": "TARGET2", "calendar_name": "TARGET2 (ECB)", "timezone": "Europe/Frankfurt"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(defaultCalendars)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendars)
}

func (h *ReportScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolveTenantID(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeLegacyScheduleGone(w)
}

type CreateScheduleRequest struct {
	ScheduleName        string `json:"schedule_name"`
	CronExpression      string `json:"cron_expression"`
	Region              string `json:"region"`
	CalendarCode        string `json:"calendar_code"`
	UnscheduledBehavior string `json:"unscheduled_behavior"`
	BusinessDayOffset   int    `json:"business_day_offset"`
	BurstDimension      string `json:"burst_dimension"`
	ExportFormat        string `json:"export_format"`
	NotifyInApp         bool   `json:"notify_in_app"`
	NotifyEmail         bool   `json:"notify_email"`
}

func (h *ReportScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolveTenantID(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeLegacyScheduleGone(w)
}

func (h *ReportScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolveTenantID(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeLegacyScheduleGone(w)
}

func (h *ReportScheduleHandler) TriggerScheduleRun(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolveTenantID(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeLegacyScheduleGone(w)
}

func (h *ReportScheduleHandler) ListScheduleBatches(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolveTenantID(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeLegacyScheduleGone(w)
}

func (h *ReportScheduleHandler) GetBatchTelemetry(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolveTenantID(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeLegacyScheduleGone(w)
}

func (h *ReportScheduleHandler) RetryBatchDLQ(w http.ResponseWriter, r *http.Request) {
	if _, err := h.resolveTenantID(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeLegacyScheduleGone(w)
}

type SyncCalendarRequest struct {
	ProviderName string `json:"provider_name"`
	FeedURL      string `json:"feed_url"`
}

func (h *ReportScheduleHandler) SyncCalendarHolidays(w http.ResponseWriter, r *http.Request) {
	calendarIDStr := chi.URLParam(r, "id")
	calendarID, err := uuid.Parse(calendarIDStr)
	if err != nil {
		http.Error(w, "Invalid Calendar ID", http.StatusBadRequest)
		return
	}

	tenantID, err := h.resolveTenantID(r)
	if err != nil {
		err = h.db.GetContext(r.Context(), &tenantID, `SELECT tenant_id FROM public.tenant_exchange_calendars WHERE id = $1`, calendarID)
		if err != nil {
			http.Error(w, "unauthorized: missing or invalid tenant identification", http.StatusUnauthorized)
			return
		}
	}

	var req SyncCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	daemon := reporting.NewHolidaySyncDaemon(h.db)
	count, err := daemon.SyncProviderHolidays(r.Context(), tenantID, calendarID, req.ProviderName, req.FeedURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "Holidays synced successfully",
		"holidays_count": count,
	})
}
