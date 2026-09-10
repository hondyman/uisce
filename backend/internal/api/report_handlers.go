package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/reports"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type ReportHandler struct {
	service *reports.ReportService
}

func NewReportHandler(service *reports.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Get("/", h.ListTemplates)
		r.Post("/", h.CreateTemplate)
		r.Get("/{id}", h.GetTemplate)
		r.Put("/{id}", h.UpdateTemplate)
		r.Patch("/{id}", h.UpdateTemplate)
		r.Delete("/{id}", h.DeleteTemplate)
		r.Put("/{id}/favorite", h.SetFavorite)
		r.Delete("/{id}/favorite", h.RemoveFavorite)
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

	// 4. Request header fallback if set by upstream auth/proxy
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
	if !isAdmin {
		if adminHeader := r.Header.Get("X-Admin"); adminHeader == "true" || adminHeader == "1" {
			isAdmin = true
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

	templates, err := h.service.ListTemplatesScoped(r.Context(), tenantID, userID)
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

