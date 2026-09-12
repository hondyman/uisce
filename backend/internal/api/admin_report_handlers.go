package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/security"

	"github.com/hondyman/uisce/backend/internal/reports"
)

type AdminReportHandler struct {
	db            *sql.DB
	adminDB       *sql.DB
	adminRepo     *reports.AdminExecutionRepository
	auditLog      AuditLogger
}

type AuditLogger interface {
	Log(ctx context.Context, tenantID, actorID, action, resourceType, resourceID string, filters map[string]interface{}) error
}

func NewAdminReportHandler(db, adminDB *sql.DB) *AdminReportHandler {
	if adminDB == nil {
		adminDB = constructAdminDB()
	}
	repo := reports.NewAdminExecutionRepository(adminDB)
	return &AdminReportHandler{
		db:        db,
		adminDB:   adminDB,
		adminRepo: repo,
		auditLog:  &dbAuditLogger{db: db},
	}
}

func NewAdminReportHandlerWithAudit(db, adminDB *sql.DB, audit AuditLogger) *AdminReportHandler {
	if adminDB == nil {
		adminDB = constructAdminDB()
	}
	repo := reports.NewAdminExecutionRepository(adminDB)
	return &AdminReportHandler{
		db:        db,
		adminDB:   adminDB,
		adminRepo: repo,
		auditLog:  audit,
	}
}

type dbAuditLogger struct {
	db *sql.DB
}

func (l *dbAuditLogger) Log(ctx context.Context, tenantID, actorID, action, resourceType, resourceID string, filters map[string]interface{}) error {
	return db.WithTenantTransaction(ctx, l.db, tenantID, func(tx *sql.Tx) error {
		filtersJSON, _ := json.Marshal(filters)
		_, err := tx.ExecContext(ctx, `
			INSERT INTO public.admin_audit_logs (tenant_id, actor_id, action, workflow_id, input, status, created_at)
			VALUES ($1, $2, $3, $4, $5, 'success', NOW())
		`, tenantID, actorID, action, resourceID, filtersJSON)
		return err
	})
}

func constructAdminDB() *sql.DB {
	dsn := os.Getenv("ADMIN_READ_DSN")
	if dsn == "" {
		return nil
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return db
}

func (h *AdminReportHandler) isConfigured() bool {
	return h.adminDB != nil
}

func (h *AdminReportHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/admin/report-executions", func(ar chi.Router) {
		ar.Get("/", h.ListExecutions)
		ar.Get("/{id}", h.GetExecution)
	})
	r.Get("/api/v1/admin/report-execution-events", h.ListEvents)
}

func (h *AdminReportHandler) checkAdmin(w http.ResponseWriter, r *http.Request) (string, bool) {
	if !h.isConfigured() {
		http.Error(w, "admin endpoint not configured", http.StatusServiceUnavailable)
		return "", false
	}
	if !adminHasRole(r, "global_admin") {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return "", false
	}
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return "", false
	}
	return auth.UserID, true
}

func (h *AdminReportHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkAdmin(w, r)
	if !ok {
		return
	}

	auditTenantID := ""
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && len(auth.TenantIDs) > 0 {
		auditTenantID = auth.TenantIDs[0]
	}

	if err := h.auditLog.Log(r.Context(), auditTenantID, userID, "monitoring.read.cross_tenant", "report_execution", "", map[string]interface{}{
		"tenant_id": r.URL.Query().Get("tenant_id"),
		"status":    r.URL.Query().Get("status"),
		"from":      r.URL.Query().Get("from"),
		"to":        r.URL.Query().Get("to"),
		"limit":     r.URL.Query().Get("limit"),
		"cursor":    r.URL.Query().Get("cursor"),
	}); err != nil {
		http.Error(w, "audit log write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var tenantID *uuid.UUID
	if tid := r.URL.Query().Get("tenant_id"); tid != "" {
		if parsed, err := uuid.Parse(tid); err == nil {
			tenantID = &parsed
		}
	}

	status := r.URL.Query().Get("status")

	var from, to *time.Time
	if f := r.URL.Query().Get("from"); f != "" {
		if parsed, err := time.Parse(time.RFC3339, f); err == nil {
			from = &parsed
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = &parsed
		}
	}

	cursorStr := r.URL.Query().Get("cursor")
	var cursor *reports.Cursor
	if cursorStr != "" {
		c, err := reports.DecodeCursor(cursorStr)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		cursor = &c
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	execs, nextCursor, err := h.adminRepo.ListExecutions(r.Context(), tenantID, status, from, to, cursor, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(execs))
	for _, e := range execs {
		item := map[string]interface{}{
			"id": e.ID, "tenant_id": e.TenantID, "template_id": e.TemplateID,
			"report_key": e.ReportKey, "status": e.Status,
			"created_at": e.CreatedAt,
		}
		if e.ScheduleID != nil {
			item["schedule_id"] = e.ScheduleID
		}
		if !e.IsPersonal && e.OutputURL.Valid && e.OutputURL.String != "" {
			item["output_url"] = e.OutputURL.String
		}
		if e.OutputSizeBytes.Valid {
			item["output_size_bytes"] = e.OutputSizeBytes.Int64
		}
		if e.ErrorMessage.Valid && e.ErrorMessage.String != "" {
			item["error_message"] = e.ErrorMessage.String
		}
		if e.RowsProcessed.Valid {
			item["rows_processed"] = e.RowsProcessed.Int64
		}
		if e.ExecutionTimeMS.Valid {
			item["execution_time_ms"] = e.ExecutionTimeMS.Int64
		}
		if e.RequestedBy.Valid {
			item["requested_by"] = e.RequestedBy.String
		}
		if e.TriggeredBy.Valid {
			item["triggered_by"] = e.TriggeredBy.String
		}
		if e.WorkflowID.Valid {
			item["workflow_id"] = e.WorkflowID.String
		}
		if e.RunID.Valid {
			item["run_id"] = e.RunID.String
		}
		if e.CompletedAt != nil {
			item["completed_at"] = e.CompletedAt
		}
		if e.Metadata != nil {
			item["metadata"] = e.Metadata
		}

		if e.IsPersonal {
			item["parameters_redacted"] = true
		} else if e.Parameters != nil {
			item["parameters"] = e.Parameters
		}

		items = append(items, item)
	}

	resp := map[string]interface{}{"items": items}
	if nextCursor != nil {
		enc, _ := reports.EncodeCursor(*nextCursor)
		resp["next_cursor"] = enc
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AdminReportHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkAdmin(w, r)
	if !ok {
		return
	}

	execIDStr := chi.URLParam(r, "id")
	execID, err := uuid.Parse(execIDStr)
	if err != nil {
		http.Error(w, "invalid execution ID", http.StatusBadRequest)
		return
	}

	exec, err := h.adminRepo.GetExecution(r.Context(), execID)
	if err != nil {
		if err == reports.ErrNotFound {
			http.Error(w, "execution not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	auditTenantID := ""
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && len(auth.TenantIDs) > 0 {
		auditTenantID = auth.TenantIDs[0]
	}

	if err := h.auditLog.Log(r.Context(), auditTenantID, userID, "monitoring.read.cross_tenant", "report_execution", execID.String(), nil); err != nil {
		http.Error(w, "audit log write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	item := map[string]interface{}{
		"id": exec.ID, "tenant_id": exec.TenantID, "template_id": exec.TemplateID,
		"report_key": exec.ReportKey, "status": exec.Status,
		"created_at": exec.CreatedAt,
	}
	if exec.ScheduleID != nil {
		item["schedule_id"] = exec.ScheduleID
	}
	if !exec.IsPersonal && exec.OutputURL.Valid && exec.OutputURL.String != "" {
		item["output_url"] = exec.OutputURL.String
	}
	if exec.OutputSizeBytes.Valid {
		item["output_size_bytes"] = exec.OutputSizeBytes.Int64
	}
	if exec.ErrorMessage.Valid && exec.ErrorMessage.String != "" {
		item["error_message"] = exec.ErrorMessage.String
	}
	if exec.RowsProcessed.Valid {
		item["rows_processed"] = exec.RowsProcessed.Int64
	}
	if exec.ExecutionTimeMS.Valid {
		item["execution_time_ms"] = exec.ExecutionTimeMS.Int64
	}
	if exec.RequestedBy.Valid {
		item["requested_by"] = exec.RequestedBy.String
	}
	if exec.TriggeredBy.Valid {
		item["triggered_by"] = exec.TriggeredBy.String
	}
	if exec.WorkflowID.Valid {
		item["workflow_id"] = exec.WorkflowID.String
	}
	if exec.RunID.Valid {
		item["run_id"] = exec.RunID.String
	}
	if exec.CompletedAt != nil {
		item["completed_at"] = exec.CompletedAt
	}
	if exec.Metadata != nil {
		item["metadata"] = exec.Metadata
	}
	if exec.IsPersonal {
		item["parameters_redacted"] = true
	} else if exec.Parameters != nil {
		item["parameters"] = exec.Parameters
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *AdminReportHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkAdmin(w, r)
	if !ok {
		return
	}

	auditTenantID := ""
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && len(auth.TenantIDs) > 0 {
		auditTenantID = auth.TenantIDs[0]
	}

	if err := h.auditLog.Log(r.Context(), auditTenantID, userID, "monitoring.read.cross_tenant", "report_execution_event", "", map[string]interface{}{
		"tenant_id": r.URL.Query().Get("tenant_id"),
		"from":      r.URL.Query().Get("from"),
		"to":        r.URL.Query().Get("to"),
		"limit":     r.URL.Query().Get("limit"),
		"cursor":    r.URL.Query().Get("cursor"),
	}); err != nil {
		http.Error(w, "audit log write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var tenantID *uuid.UUID
	if tid := r.URL.Query().Get("tenant_id"); tid != "" {
		if parsed, err := uuid.Parse(tid); err == nil {
			tenantID = &parsed
		}
	}

	cursorStr := r.URL.Query().Get("cursor")
	var cursor *reports.Cursor
	if cursorStr != "" {
		c, err := reports.DecodeCursor(cursorStr)
		if err != nil {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		cursor = &c
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, nextCursor, err := h.adminRepo.ListEvents(r.Context(), tenantID, nil, nil, cursor, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(events))
	for _, ev := range events {
		item := map[string]interface{}{
			"id": ev.ID, "execution_id": ev.ExecutionID, "tenant_id": ev.TenantID,
			"event": ev.Event, "to_status": ev.ToStatus,
			"actor_id": ev.ActorID, "created_at": ev.CreatedAt,
		}
		if ev.FromStatus.Valid {
			item["from_status"] = ev.FromStatus.String
		}
		if len(ev.Detail) > 0 {
			item["detail"] = ev.Detail
		}
		items = append(items, item)
	}

	resp := map[string]interface{}{"items": items}
	if nextCursor != nil {
		enc, _ := reports.EncodeCursor(*nextCursor)
		resp["next_cursor"] = enc
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func adminHasRole(r *http.Request, role string) bool {
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		return false
	}
	for _, authRole := range auth.Roles {
		if authRole == role {
			return true
		}
	}
	return false
}
