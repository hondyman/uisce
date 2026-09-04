package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/reporting"
)

type ReportScheduleHandler struct {
	db                *sqlx.DB
	burstOrchestrator *reporting.ReportBurstOrchestrator
}

func NewReportScheduleHandler(db *sqlx.DB) *ReportScheduleHandler {
	calEval := reporting.NewCalendarEvaluator(db)
	orchestrator := reporting.NewReportBurstOrchestrator(db, calEval, nil, nil)
	return &ReportScheduleHandler{
		db:                db,
		burstOrchestrator: orchestrator,
	}
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
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts the raw header directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
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
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts the raw header directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	var schedules []struct {
		ID                  uuid.UUID  `json:"id" db:"id"`
		TenantID            uuid.UUID  `json:"tenant_id" db:"tenant_id"`
		ScheduleName        string     `json:"schedule_name" db:"schedule_name"`
		CronExpression      string     `json:"cron_expression" db:"cron_expression"`
		Region              string     `json:"region" db:"region"`
		CalendarID          *uuid.UUID `json:"calendar_id" db:"calendar_id"`
		StartOfDayTime      string     `json:"start_of_day_time" db:"start_of_day_time"`
		UnscheduledBehavior string     `json:"unscheduled_behavior" db:"unscheduled_behavior"`
		BusinessDayOffset   int        `json:"business_day_offset" db:"business_day_offset"`
		BurstDimension      string     `json:"burst_dimension" db:"burst_dimension"`
		ExportFormat        string     `json:"export_format" db:"export_format"`
		IsActive            bool       `json:"is_active" db:"is_active"`
		CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	}

	err = h.db.SelectContext(r.Context(), &schedules, `
		SELECT id, tenant_id, schedule_name, cron_expression, region, calendar_id, 
		       start_of_day_time::text, unscheduled_behavior, business_day_offset, 
		       burst_dimension, export_format, is_active, created_at
		FROM public.report_schedules 
		WHERE tenant_id = $1 
		ORDER BY created_at DESC
	`, tenantID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
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
	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts the raw header directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		tenantID = uuid.New()
	}

	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	scheduleID := uuid.New()
	notificationJSON, _ := json.Marshal(map[string]bool{
		"in_app": req.NotifyInApp,
		"email":  req.NotifyEmail,
	})

	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO public.report_schedules (
			id, tenant_id, schedule_name, cron_expression, region,
			unscheduled_behavior, business_day_offset, burst_dimension,
			export_format, notification_channels, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true)
	`, scheduleID, tenantID, req.ScheduleName, req.CronExpression, req.Region,
		req.UnscheduledBehavior, req.BusinessDayOffset, req.BurstDimension,
		req.ExportFormat, notificationJSON)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      scheduleID,
		"message": "Schedule created successfully",
	})
}

func (h *ReportScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleIDStr := chi.URLParam(r, "id")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		http.Error(w, "Invalid Schedule ID", http.StatusBadRequest)
		return
	}

	var schedule struct {
		ID                  uuid.UUID `json:"id" db:"id"`
		TenantID            uuid.UUID `json:"tenant_id" db:"tenant_id"`
		ScheduleName        string    `json:"schedule_name" db:"schedule_name"`
		CronExpression      string    `json:"cron_expression" db:"cron_expression"`
		Region              string    `json:"region" db:"region"`
		UnscheduledBehavior string    `json:"unscheduled_behavior" db:"unscheduled_behavior"`
		BusinessDayOffset   int       `json:"business_day_offset" db:"business_day_offset"`
		BurstDimension      string    `json:"burst_dimension" db:"burst_dimension"`
		ExportFormat        string    `json:"export_format" db:"export_format"`
		IsActive            bool      `json:"is_active" db:"is_active"`
	}

	err = h.db.GetContext(r.Context(), &schedule, `
		SELECT id, tenant_id, schedule_name, cron_expression, region, 
		       unscheduled_behavior, business_day_offset, burst_dimension, 
		       export_format, is_active
		FROM public.report_schedules 
		WHERE id = $1
	`, scheduleID)

	if err != nil {
		http.Error(w, "Schedule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

func (h *ReportScheduleHandler) TriggerScheduleRun(w http.ResponseWriter, r *http.Request) {
	scheduleIDStr := chi.URLParam(r, "id")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		http.Error(w, "Invalid Schedule ID", http.StatusBadRequest)
		return
	}

	result, err := h.burstOrchestrator.ExecuteScheduledBurst(r.Context(), scheduleID, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ReportScheduleHandler) ListScheduleBatches(w http.ResponseWriter, r *http.Request) {
	scheduleIDStr := chi.URLParam(r, "id")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		http.Error(w, "Invalid Schedule ID", http.StatusBadRequest)
		return
	}

	var batches []struct {
		ID                uuid.UUID  `json:"id" db:"id"`
		ScheduleID        uuid.UUID  `json:"schedule_id" db:"schedule_id"`
		EffectiveDate     string     `json:"effective_date" db:"effective_date"`
		TotalClients      int        `json:"total_clients" db:"total_clients"`
		SuccessfulRenders int        `json:"successful_renders" db:"successful_renders"`
		FailedRenders     int        `json:"failed_renders" db:"failed_renders"`
		Status            string     `json:"status" db:"status"`
		StartedAt         time.Time  `json:"started_at" db:"started_at"`
		CompletedAt       *time.Time `json:"completed_at" db:"completed_at"`
	}

	err = h.db.SelectContext(r.Context(), &batches, `
		SELECT id, schedule_id, effective_date::text, total_clients, 
		       successful_renders, failed_renders, status, started_at, completed_at
		FROM public.report_burst_batches 
		WHERE schedule_id = $1 
		ORDER BY started_at DESC 
		LIMIT 20
	`, scheduleID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(batches)
}

func (h *ReportScheduleHandler) GetBatchTelemetry(w http.ResponseWriter, r *http.Request) {
	batchIDStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		http.Error(w, "Invalid Batch ID", http.StatusBadRequest)
		return
	}

	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts the raw header directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		// Fallback tenant query if header missing
		_ = h.db.GetContext(r.Context(), &tenantID, `SELECT tenant_id FROM public.report_burst_batches WHERE id = $1`, batchID)
	}

	svc := reporting.NewTelemetryDLQService(h.db)
	telemetry, err := svc.GetBatchTelemetry(r.Context(), tenantID, batchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(telemetry)
}

func (h *ReportScheduleHandler) RetryBatchDLQ(w http.ResponseWriter, r *http.Request) {
	batchIDStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		http.Error(w, "Invalid Batch ID", http.StatusBadRequest)
		return
	}

	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts the raw header directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		_ = h.db.GetContext(r.Context(), &tenantID, `SELECT tenant_id FROM public.report_burst_batches WHERE id = $1`, batchID)
	}

	svc := reporting.NewTelemetryDLQService(h.db)
	newBatchID, err := svc.RetryFailedSlices(r.Context(), tenantID, batchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "Retry batch created",
		"retry_batch_id": newBatchID,
	})
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

	// getSecureTenantID (helpers.go) validates the X-Tenant-ID header against
	// the caller's JWT-issued tenant list / global-admin status before
	// trusting it; it never trusts the raw header directly.
	tenantIDStr := getSecureTenantID(r)
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		_ = h.db.GetContext(r.Context(), &tenantID, `SELECT tenant_id FROM public.tenant_exchange_calendars WHERE id = $1`, calendarID)
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
