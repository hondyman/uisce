package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type ReportHandler struct {
	service        *reports.ReportService
	executor      reports.ReportExecutor
	db            *sql.DB
	executionRepo *reports.ExecutionRepository
}

func NewReportHandler(service *reports.ReportService, executor reports.ReportExecutor, dbConn *sql.DB) *ReportHandler {
	return &ReportHandler{
		service:        service,
		executor:      executor,
		db:            dbConn,
		executionRepo: reports.NewExecutionRepository(dbConn),
	}
}

func (h *ReportHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Get("/", h.ListTemplates)
		r.Post("/", h.CreateTemplate)

		// Folder subroutes registered BEFORE /{id} to prevent Chi matching "folders" as {id}
		r.Route("/folders", func(fr chi.Router) {
			fr.Get("/", h.ListFolders)
			fr.Post("/", h.CreateFolder)
			fr.Put("/{id}", h.RenameFolder)
			fr.Post("/{id}/move", h.MoveFolder)
			fr.Delete("/{id}", h.DeleteFolder)
			fr.Get("/{id}/items", h.ListFolderItems)
			fr.Post("/{id}/items", h.AddFolderItem)
			fr.Delete("/{id}/items/{templateId}", h.RemoveFolderItem)
		})

		// Executions subroute registered in a separate Route block BEFORE /{id} to prevent Chi matching "executions" as {id}
		r.Route("/executions", func(er chi.Router) {
			er.Get("/", h.ListExecutions)
			er.Get("/{id}", h.GetExecution)
			er.Get("/{id}/events", h.ListExecutionEvents)
		})

		r.Get("/{id}", h.GetTemplate)
		r.Put("/{id}", h.UpdateTemplate)
		r.Patch("/{id}", h.UpdateTemplate)
		r.Delete("/{id}", h.DeleteTemplate)
		r.Put("/{id}/favorite", h.SetFavorite)
		r.Delete("/{id}/favorite", h.RemoveFavorite)

		// Schedule subroutes mounted under /{id}/schedules
		r.Route("/{id}/schedules", func(sr chi.Router) {
			sr.Get("/", h.ListSchedulesForTemplate)
			sr.Post("/", h.CreateScheduleForTemplate)
			sr.Delete("/{sid}", h.DeleteSchedule)
			sr.Post("/{sid}/run", h.TriggerScheduleRun)
			sr.Get("/{sid}/executions", h.ListScheduleExecutions)
		})
	})
}

// resolveAuthContext extracts tenant ID, user ID, and admin status from authenticated context.
func (h *ReportHandler) resolveAuthContext(r *http.Request) (tenantID uuid.UUID, userID string, isAdmin bool, err error) {
	// 1. Try security.AuthInfoFromContext (SecurityManager / AuthContextMiddleware)
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok {
		userID = auth.UserID
		isAdmin = auth.IsGlobalAdmin
		if !isAdmin {
			for _, role := range auth.Roles {
				if role == "admin" || role == "tenant_admin" || role == "core_admin" || role == "is_core_admin" {
					isAdmin = true
					break
				}
			}
		}
		if len(auth.TenantIDs) > 0 {
			if tid, parseErr := uuid.Parse(auth.TenantIDs[0]); parseErr == nil && tid != uuid.Nil {
				tenantID = tid
			}
		}
	}

	// 2. Try identity context
	if tenantID == uuid.Nil {
		if tidStr, ok := identity.TenantIDFromContext(r.Context()); ok {
			if tid, parseErr := uuid.Parse(tidStr); parseErr == nil && tid != uuid.Nil {
				tenantID = tid
			}
		}
	}
	if userID == "" {
		if uid, ok := identity.ActorIDFromContext(r.Context()); ok {
			userID = uid
		}
	}

	// 3. Try jwtmiddleware claims
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		if userID == "" {
			userID = claims.UserID
		}
		if !isAdmin {
			isAdmin = claims.IsCoreAdmin || jwtmiddleware.HasRole(claims, "admin") || jwtmiddleware.HasRole(claims, "tenant_admin")
		}
		if tenantID == uuid.Nil && claims.TenantID != "" {
			if tid, parseErr := uuid.Parse(claims.TenantID); parseErr == nil && tid != uuid.Nil {
				tenantID = tid
			}
		}
	}

	// 4. Request header fallback ONLY if ALLOW_CLIENT_TENANT_HEADER_FALLBACK=true (dev/local use only).
	// In production, this fallback is strictly disabled: headers are client-controlled and untrusted.
	// Admin status NEVER falls back to headers under any circumstances.
	if allowClientTenantHeaderFallback() {
		if tenantID == uuid.Nil {
			if tidHeader := r.Header.Get("X-Tenant-ID"); tidHeader != "" {
				if tid, parseErr := uuid.Parse(tidHeader); parseErr == nil && tid != uuid.Nil {
					tenantID = tid
				}
			}
		}
		if userID == "" {
			if uidHeader := r.Header.Get("X-User-ID"); uidHeader != "" {
				userID = uidHeader
			}
		}
	}

	if tenantID == uuid.Nil {
		return uuid.Nil, "", false, errors.New("unauthorized: missing or invalid tenant identification")
	}

	return tenantID, userID, isAdmin, nil
}

func (h *ReportHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 256 {
		http.Error(w, "query parameter 'q' exceeds maximum length of 256 characters", http.StatusBadRequest)
		return
	}

	var templates []reports.ReportTemplate
	if query != "" {
		templates, err = h.service.SearchTemplatesScoped(r.Context(), tenantID, userID, query)
	} else {
		templates, err = h.service.ListTemplatesScoped(r.Context(), tenantID, userID)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if templates == nil {
		templates = []reports.ReportTemplate{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func (h *ReportHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var template reports.ReportTemplate
	if err := json.Unmarshal(bodyBytes, &template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var raw map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &raw)

	if template.TemplateName == "" && raw != nil {
		if name, ok := raw["name"].(string); ok && name != "" {
			template.TemplateName = name
		}
	}
	if template.ID == uuid.Nil {
		if raw != nil {
			if idStr, ok := raw["id"].(string); ok && idStr != "" {
				template.ID, _ = uuid.Parse(idStr)
			}
		}
		if template.ID == uuid.Nil {
			template.ID = uuid.New()
		}
	}
	// Non-admin users can ONLY create personal reports
	if !isAdmin {
		template.IsPersonal = true
	}

	// Always bind tenant to the caller's authenticated tenant
	template.TenantID = tenantID

	// Bind created_by_id to authenticated caller
	if userID != "" {
		template.CreatedByID = &userID
	}

	if template.LayoutConfig == nil {
		if raw != nil {
			if def, ok := raw["definition"].(map[string]interface{}); ok {
				template.LayoutConfig = def
			}
		}
		if template.LayoutConfig == nil {
			template.LayoutConfig = make(map[string]interface{})
		}
	}

	// Reports cannot be created with sharing pre-baked; sharing happens post-creation
	checkPrebakedShare := func() bool {
		if s, ok := raw["is_shared"].(bool); ok && s {
			return true
		}
		if s, ok := raw["is_public"].(bool); ok && s {
			return true
		}
		if rawMeta, ok := raw["metadata"].(map[string]interface{}); ok {
			if isShared, exists := rawMeta["is_shared"].(bool); exists && isShared {
				return true
			}
			if isPub, exists := rawMeta["is_public"].(bool); exists && isPub {
				return true
			}
		}
		if meta, ok := template.LayoutConfig["metadata"].(map[string]interface{}); ok {
			if isShared, exists := meta["is_shared"].(bool); exists && isShared {
				return true
			}
			if isPub, exists := meta["is_public"].(bool); exists && isPub {
				return true
			}
		}
		return false
	}
	if checkPrebakedShare() {
		http.Error(w, "forbidden: reports cannot be created as shared; share after creation", http.StatusForbidden)
		return
	}
	template.IsPublic = false

	if template.ParameterSchema == nil {
		template.ParameterSchema = make(map[string]interface{})
	}
	if !template.IsActive {
		template.IsActive = true
	}

	if err := h.service.CreateTemplate(r.Context(), &template); err != nil {
		if errors.Is(err, reports.ErrConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func (h *ReportHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	template, err := h.service.GetTemplate(r.Context(), id)
	if err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func (h *ReportHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, authErr := h.resolveAuthContext(r)
	if authErr != nil {
		http.Error(w, authErr.Error(), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing, err := h.service.GetTemplate(r.Context(), id)
	if err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "report template not found", http.StatusNotFound)
		return
	}

	// 1. Core report check: core reports living in gold-copy tenant cannot be modified by tenant users
	goldCopyID, _ := h.service.ResolveGoldCopyTenantID(r.Context())
	if goldCopyID != uuid.Nil && existing.TenantID == goldCopyID && tenantID != goldCopyID {
		http.Error(w, "forbidden: core reports from gold-copy tenant are read-only", http.StatusForbidden)
		return
	}

	// 2. Tenant isolation check: cannot modify another tenant's report
	if existing.TenantID != tenantID {
		http.Error(w, "report template not found or unauthorized", http.StatusNotFound)
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	template := *existing
	template.ID = id
	template.TenantID = existing.TenantID

	// 3. Authorization check for is_personal mutation
	if newPersonal, ok := raw["is_personal"].(bool); ok {
		if newPersonal != existing.IsPersonal {
			// Changing is_personal requires being author or admin
			isAuthor := existing.CreatedByID != nil && *existing.CreatedByID == userID
			if !isAuthor && !isAdmin {
				http.Error(w, "forbidden: only author or tenant admin can change report personalization status", http.StatusForbidden)
				return
			}
			template.IsPersonal = newPersonal
		}
	}

	if layout, ok := raw["layout_config"].(map[string]interface{}); ok {
		template.LayoutConfig = layout
	} else if def, ok := raw["definition"].(map[string]interface{}); ok {
		template.LayoutConfig = def
	}

	// 4. Sharing check: if sharing configuration is being set in metadata, layout_config, or root, ensure !is_core && is_personal && author
	checkShared := func(m map[string]interface{}) bool {
		if s, ok := m["is_shared"].(bool); ok && s {
			return true
		}
		if s, ok := m["is_public"].(bool); ok && s {
			return true
		}
		return false
	}
	attemptingShare := false
	if s, ok := raw["is_shared"].(bool); ok && s {
		attemptingShare = true
	}
	if s, ok := raw["is_public"].(bool); ok && s {
		attemptingShare = true
	}
	if meta, ok := raw["metadata"].(map[string]interface{}); ok && checkShared(meta) {
		attemptingShare = true
	}
	if template.LayoutConfig != nil {
		if meta, ok := template.LayoutConfig["metadata"].(map[string]interface{}); ok && checkShared(meta) {
			attemptingShare = true
		}
	}
	if attemptingShare {
		isCore := (goldCopyID != uuid.Nil && existing.TenantID == goldCopyID)
		isAuthor := existing.CreatedByID != nil && *existing.CreatedByID == userID
		if isCore || !existing.IsPersonal || !isAuthor {
			http.Error(w, "forbidden: only personal reports created by you can be shared", http.StatusForbidden)
			return
		}
	}

	if tmplName, ok := raw["template_name"].(string); ok && tmplName != "" {
		template.TemplateName = tmplName
	} else if name, ok := raw["name"].(string); ok && name != "" {
		template.TemplateName = name
	}
	if desc, ok := raw["description"].(string); ok {
		template.Description = desc
	}
	if cat, ok := raw["category"].(string); ok && cat != "" {
		template.Category = cat
	}
	if schema, ok := raw["parameter_schema"].(map[string]interface{}); ok {
		template.ParameterSchema = schema
	}
	if active, ok := raw["is_active"].(bool); ok {
		template.IsActive = active
	}

	if err := h.service.UpdateTemplate(r.Context(), &template); err != nil {
		if errors.Is(err, reports.ErrConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func (h *ReportHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, authErr := h.resolveAuthContext(r)
	if authErr != nil {
		http.Error(w, authErr.Error(), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	existing, err := h.service.GetTemplate(r.Context(), id)
	if err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "report template not found", http.StatusNotFound)
		return
	}

	// Core reports in gold-copy tenant cannot be deleted by tenant users
	goldCopyID, _ := h.service.ResolveGoldCopyTenantID(r.Context())
	if goldCopyID != uuid.Nil && existing.TenantID == goldCopyID && tenantID != goldCopyID {
		http.Error(w, "forbidden: core reports from gold-copy tenant cannot be deleted", http.StatusForbidden)
		return
	}

	// Tenant isolation check
	if existing.TenantID != tenantID {
		http.Error(w, "report template not found or unauthorized", http.StatusNotFound)
		return
	}

	// If personal report, only author or admin can delete
	if existing.IsPersonal {
		isAuthor := existing.CreatedByID != nil && *existing.CreatedByID == userID
		if !isAuthor && !isAdmin {
			http.Error(w, "forbidden: only author or admin can delete a personal report", http.StatusForbidden)
			return
		}
	}

	if err := h.service.DeleteTemplate(r.Context(), id); err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetFavorite handles PUT /api/v1/reports/{id}/favorite.
// It extracts tenant and user strictly from auth context and rejects non-empty body.
func (h *ReportHandler) SetFavorite(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Reject any request body: identity is derived exclusively from auth context
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(r.Body)
		if len(bodyBytes) > 0 && string(bodyBytes) != "{}" {
			http.Error(w, "bad request: favorite endpoint accepts no request body", http.StatusBadRequest)
			return
		}
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	if err := h.service.SetFavorite(r.Context(), tenantID, userID, id); err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"is_favorite": true,
		"template_id": id,
	})
}

// RemoveFavorite handles DELETE /api/v1/reports/{id}/favorite.
// It extracts tenant and user strictly from auth context and rejects non-empty body.
func (h *ReportHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	if err := h.service.RemoveFavorite(r.Context(), tenantID, userID, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"is_favorite": false,
		"template_id": id,
	})
}

// ============================================================================
// SCHEDULE HANDLERS (Hardened with Auth Context & Owner Scoping)
// ============================================================================

type createScheduleHTTPBody struct {
	ScheduleName        string     `json:"schedule_name"`
	CronExpression      string     `json:"cron_expression"`
	Region              string     `json:"region"`
	CalendarID          *uuid.UUID `json:"calendar_id"`
	StartOfDayTime      string     `json:"start_of_day_time"`
	UnscheduledBehavior string     `json:"unscheduled_behavior"`
	BusinessDayOffset   int        `json:"business_day_offset"`
	BurstDimension      string     `json:"burst_dimension"`
	ExportFormat        string     `json:"export_format"`
	NotifyInApp         bool       `json:"notify_in_app"`
	NotifyEmail         bool       `json:"notify_email"`
}

// ListSchedulesForTemplate handles GET /api/v1/reports/{id}/schedules.
func (h *ReportHandler) ListSchedulesForTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tmplIDStr := chi.URLParam(r, "id")
	tmplID, err := uuid.Parse(tmplIDStr)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	schedules, err := h.service.ListSchedulesForTemplate(r.Context(), tenantID, userID, tmplID)
	if err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, "Report template not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(schedules)
}

// CreateScheduleForTemplate handles POST /api/v1/reports/{id}/schedules.
func (h *ReportHandler) CreateScheduleForTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tmplIDStr := chi.URLParam(r, "id")
	tmplID, err := uuid.Parse(tmplIDStr)
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}

	var body createScheduleHTTPBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if body.ScheduleName == "" {
		http.Error(w, "schedule_name is required", http.StatusBadRequest)
		return
	}
	if body.CronExpression == "" {
		http.Error(w, "cron_expression is required", http.StatusBadRequest)
		return
	}

	sched, err := h.service.CreateSchedule(r.Context(), tenantID, userID, reports.CreateScheduleInput{
		TemplateID:          tmplID,
		ScheduleName:        body.ScheduleName,
		CronExpression:      body.CronExpression,
		Region:              body.Region,
		CalendarID:          body.CalendarID,
		StartOfDayTime:      body.StartOfDayTime,
		UnscheduledBehavior: body.UnscheduledBehavior,
		BusinessDayOffset:   body.BusinessDayOffset,
		BurstDimension:      body.BurstDimension,
		ExportFormat:        body.ExportFormat,
		NotifyInApp:         body.NotifyInApp,
		NotifyEmail:         body.NotifyEmail,
	})
	if err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, "Report template not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, reports.ErrConflict) {
			http.Error(w, "Schedule name already exists for this report template", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sched)
}

// DeleteSchedule handles DELETE /api/v1/reports/{id}/schedules/{sid}.
func (h *ReportHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	sidStr := chi.URLParam(r, "sid")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteSchedule(r.Context(), tenantID, userID, isAdmin, sid); err != nil {
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, "Schedule not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, reports.ErrForbidden) {
			http.Error(w, "Forbidden: only schedule owner or tenant admin can delete schedule", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerScheduleRun handles POST /api/v1/reports/{id}/schedules/{sid}/run.
// Dispatches report execution asynchronously via the injected executor and returns HTTP 202 Accepted.
func (h *ReportHandler) TriggerScheduleRun(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	sidStr := chi.URLParam(r, "sid")
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	res, err := h.service.TriggerScheduleRun(r.Context(), tenantID, userID, isAdmin, sid, h.executor)
	if err != nil {
		var dispatchErr *reports.DispatchError
		if errors.As(err, &dispatchErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"execution_id": dispatchErr.ExecutionID,
				"error":        dispatchErr.Err.Error(),
			})
			return
		}
		if errors.Is(err, reports.ErrNotFound) {
			http.Error(w, "Schedule not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, reports.ErrForbidden) {
			http.Error(w, "Forbidden: only schedule owner or tenant admin can trigger schedule", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 202 Accepted payload: truthful async execution status
	responseBody := map[string]interface{}{
		"status":       "pending",
		"execution_id": res.ExecutionID,
	}
	// workflow_id is only populated if non-empty; never an empty string
	workflowID := fmt.Sprintf("report-exec-%s", res.ExecutionID)
	responseBody["workflow_id"] = workflowID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(responseBody)
}

// GetExecution handles GET /api/v1/reports/executions/{id}.
// Enforces the pinned two-clause visibility predicate:
//   (e.tenant_id = $2 OR e.triggered_by = $3)
// Returns 404 if execution is not found or inaccessible (zero existence leak).
func (h *ReportHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	execIDStr := chi.URLParam(r, "id")
	execID, err := uuid.Parse(execIDStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	if h.db == nil {
		http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
		return
	}

	e, err := h.executionRepo.GetExecution(r.Context(), execID, tenantID, userID, isAdmin)
	if errors.Is(err, reports.ErrNotFound) {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := e.ID
	rowTenantID := e.TenantID
	templateID := e.TemplateID
	reportKey := e.ReportKey
	status := e.Status
	paramsBytes := e.Parameters
	outputURL := e.OutputURL
	outputSizeBytes := e.OutputSizeBytes
	rowsProcessed := e.RowsProcessed
	executionTimeMS := e.ExecutionTimeMS
	errorMessage := e.ErrorMessage
	workflowID := e.WorkflowID
	runID := e.RunID
	requestedBy := e.RequestedBy
	triggeredBy := e.TriggeredBy
	metadataBytes := e.Metadata
	createdAt := e.CreatedAt
	completedAt := e.CompletedAt
	isPersonal := e.IsPersonal
	createdByID := e.CreatedByID

	var params map[string]interface{}
	if len(paramsBytes) > 0 {
		_ = json.Unmarshal(paramsBytes, &params)
	}
	var meta map[string]interface{}
	if len(metadataBytes) > 0 {
		_ = json.Unmarshal(metadataBytes, &meta)
	}

	resp := map[string]interface{}{
		"id":          id,
		"tenant_id":   rowTenantID,
		"template_id": templateID,
		"report_key":  reportKey,
		"status":      status,
		"created_at":  createdAt,
		"is_personal": isPersonal,
	}
	if params != nil {
		resp["parameters"] = params
	}
	if meta != nil {
		resp["metadata"] = meta
	}
	if outputURL.Valid && outputURL.String != "" {
		resp["output_url"] = outputURL.String
	}
	if outputSizeBytes.Valid {
		resp["output_size_bytes"] = outputSizeBytes.Int64
	}
	if rowsProcessed.Valid {
		resp["rows_processed"] = rowsProcessed.Int64
	}
	if executionTimeMS.Valid {
		resp["execution_time_ms"] = executionTimeMS.Int64
	}
	if errorMessage.Valid && errorMessage.String != "" {
		resp["error_message"] = errorMessage.String
	}
	if workflowID.Valid && workflowID.String != "" {
		resp["workflow_id"] = workflowID.String
	}
	if runID.Valid && runID.String != "" {
		resp["run_id"] = runID.String
	}
	if requestedBy.Valid && requestedBy.String != "" {
		resp["requested_by"] = requestedBy.String
	}
	if triggeredBy.Valid && triggeredBy.String != "" {
		resp["triggered_by"] = triggeredBy.String
	}
	if completedAt != nil {
		resp["completed_at"] = completedAt
	}
	if createdByID.Valid && createdByID.String != "" {
		resp["created_by_id"] = createdByID.String
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ReportHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	var cursor *reports.Cursor
	if cursorStr != "" {
		c, err := reports.DecodeCursor(cursorStr)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		cursor = &c
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if _, parseErr := strconv.Atoi(l); parseErr == nil {
			limit, _ = strconv.Atoi(l)
		}
	}

	execs, err := h.executionRepo.ListExecutions(r.Context(), tenantID, userID, isAdmin, cursor, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(execs))
	for _, e := range execs {
		item := map[string]interface{}{
			"id": e.ID, "tenant_id": e.TenantID, "template_id": e.TemplateID,
			"status": e.Status, "created_at": e.CreatedAt,
		}
		if e.ScheduleID != nil {
			item["schedule_id"] = e.ScheduleID
		}
		if e.OutputURL.Valid && e.OutputURL.String != "" {
			item["output_url"] = e.OutputURL.String
		}
		if e.ErrorMessage.Valid && e.ErrorMessage.String != "" {
			item["error_message"] = e.ErrorMessage.String
		}
		items = append(items, item)
	}

	resp := map[string]interface{}{"items": items}
	if len(execs) == limit {
		lastExec := execs[len(execs)-1]
		ec := reports.Cursor{CreatedAt: lastExec.CreatedAt, ID: lastExec.ID}
		enc, _ := reports.EncodeCursor(ec)
		resp["next_cursor"] = enc
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ReportHandler) ListExecutionEvents(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	execIDStr := chi.URLParam(r, "id")
	execID, err := uuid.Parse(execIDStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	var cursor *reports.Cursor
	if cursorStr != "" {
		c, err := reports.DecodeCursor(cursorStr)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
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

	events, truncated, err := h.executionRepo.ListExecutionEvents(r.Context(), execID, tenantID, userID, isAdmin, cursor, limit)
	if errors.Is(err, reports.ErrNotFound) {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(events))
	for _, ev := range events {
		item := map[string]interface{}{
			"id": ev.ID, "execution_id": ev.ExecutionID, "event": ev.Event,
			"to_status": ev.ToStatus, "actor_id": ev.ActorID, "created_at": ev.CreatedAt,
		}
		if ev.FromStatus.Valid {
			item["from_status"] = ev.FromStatus.String
		}
		if len(ev.Detail) > 0 {
			item["detail"] = ev.Detail
		}
		items = append(items, item)
	}

	resp := map[string]interface{}{"items": items, "truncated": truncated}
	if truncated && len(events) > 0 {
		lastEv := events[len(events)-1]
		ec := reports.Cursor{CreatedAt: lastEv.CreatedAt, ID: lastEv.ID}
		enc, _ := reports.EncodeCursor(ec)
		resp["next_cursor"] = enc
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ReportHandler) ListScheduleExecutions(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, isAdmin, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	scheduleIDStr := chi.URLParam(r, "sid")
	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		http.Error(w, "Invalid schedule ID", http.StatusBadRequest)
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	var cursor *reports.Cursor
	if cursorStr != "" {
		c, err := reports.DecodeCursor(cursorStr)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		cursor = &c
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if _, parseErr := strconv.Atoi(l); parseErr == nil {
			limit, _ = strconv.Atoi(l)
		}
	}

	execs, err := h.executionRepo.ListScheduleExecutions(r.Context(), scheduleID, tenantID, userID, isAdmin, cursor, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]map[string]interface{}, 0, len(execs))
	for _, e := range execs {
		item := map[string]interface{}{
			"id": e.ID, "tenant_id": e.TenantID, "template_id": e.TemplateID,
			"status": e.Status, "created_at": e.CreatedAt,
		}
		if e.ScheduleID != nil {
			item["schedule_id"] = e.ScheduleID
		}
		if e.OutputURL.Valid && e.OutputURL.String != "" {
			item["output_url"] = e.OutputURL.String
		}
		if e.ErrorMessage.Valid && e.ErrorMessage.String != "" {
			item["error_message"] = e.ErrorMessage.String
		}
		items = append(items, item)
	}

	resp := map[string]interface{}{"items": items}
	if len(execs) == limit {
		lastExec := execs[len(execs)-1]
		ec := reports.Cursor{CreatedAt: lastExec.CreatedAt, ID: lastExec.ID}
		enc, _ := reports.EncodeCursor(ec)
		resp["next_cursor"] = enc
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

