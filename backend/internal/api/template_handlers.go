package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ============================================================================
// Template Handler - All API Endpoints for Semantic Query Templates
// ============================================================================

type TemplateHandler struct {
	store   *TemplateStore
	gateway *LLMGateway
	rbac    *TemplateRBAC
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(store *TemplateStore, gateway *LLMGateway) *TemplateHandler {
	return &TemplateHandler{
		store:   store,
		gateway: gateway,
	}
}

// ============================================================================
// POST /api/semantic/templates - Create Template
// ============================================================================

// CreateTemplateRequest is the request body for creating a template
type CreateTemplateRequest struct {
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	Datasource    string             `json:"datasource"`
	Version       string             `json:"version"`
	SemanticQuery *SemanticQuery     `json:"semantic_query"`
	Parameters    []TemplateParamDef `json:"parameters,omitempty"`
	Visibility    string             `json:"visibility"` // "private", "team", "public"
	Tags          []string           `json:"tags,omitempty"`
}

func (h *TemplateHandler) HandleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")

	if tenantID == "" || userID == "" {
		http.Error(w, "X-Tenant-ID and X-User-ID headers required", http.StatusBadRequest)
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if req.Name == "" || req.Datasource == "" {
		http.Error(w, "Name and datasource are required", http.StatusBadRequest)
		return
	}

	// Extract parameters from query if not provided
	if len(req.Parameters) == 0 {
		paramNames := ExtractParametersFromQuery(req.SemanticQuery)
		for _, name := range paramNames {
			req.Parameters = append(req.Parameters, TemplateParamDef{
				Name:     name,
				Type:     ParamString,
				Required: true,
			})
		}
	}

	// Create template
	t := &SemanticQueryTemplate{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		Datasource:    req.Datasource,
		Version:       req.Version,
		SemanticQuery: req.SemanticQuery,
		Parameters:    req.Parameters,
		CreatedBy:     userID,
		Visibility:    req.Visibility,
		Tags:          req.Tags,
	}

	if err := h.store.Create(ctx, t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

// ============================================================================
// GET /api/semantic/templates/{id} - Get Template
// ============================================================================

func (h *TemplateHandler) HandleGetTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	id := chi.URLParam(r, "id")

	t, err := h.store.Get(ctx, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// Verify tenant access
	if t.TenantID != tenantID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// ============================================================================
// GET /api/semantic/templates - List Templates
// ============================================================================

func (h *TemplateHandler) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	params := &TemplateListQueryParams{
		Datasource:     r.URL.Query().Get("datasource"),
		Version:        r.URL.Query().Get("version"),
		CreatedBy:      r.URL.Query().Get("created_by"),
		Tag:            r.URL.Query().Get("tag"),
		Visibility:     r.URL.Query().Get("visibility"),
		ShowDeprecated: r.URL.Query().Get("show_deprecated") == "true",
		Limit:          limit,
		Offset:         offset,
	}

	templates, err := h.store.List(ctx, tenantID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
		"limit":     limit,
		"offset":    offset,
	})
}

// ============================================================================
// PUT /api/semantic/templates/{id} - Update Template
// ============================================================================

type UpdateTemplateRequest struct {
	Name          string             `json:"name,omitempty"`
	Description   string             `json:"description,omitempty"`
	SemanticQuery *SemanticQuery     `json:"semantic_query,omitempty"`
	Parameters    []TemplateParamDef `json:"parameters,omitempty"`
	Visibility    string             `json:"visibility,omitempty"`
	Tags          []string           `json:"tags,omitempty"`
	Deprecated    bool               `json:"deprecated,omitempty"`
	ChangeMessage string             `json:"change_message,omitempty"`
}

func (h *TemplateHandler) HandleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	id := chi.URLParam(r, "id")

	// Get existing template
	existing, err := h.store.Get(ctx, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	if existing.TenantID != tenantID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Apply updates (only non-zero values)
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.SemanticQuery != nil {
		existing.SemanticQuery = req.SemanticQuery
	}
	if len(req.Parameters) > 0 {
		existing.Parameters = req.Parameters
	}
	if req.Visibility != "" {
		existing.Visibility = req.Visibility
	}
	if len(req.Tags) > 0 {
		existing.Tags = req.Tags
	}
	if req.Deprecated {
		existing.Deprecated = true
		now := time.Now()
		existing.DeprecatedAt = &now
	}

	// Update in store
	if err := h.store.Update(ctx, id, existing, req.ChangeMessage); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

// ============================================================================
// DELETE /api/semantic/templates/{id} - Delete Template
// ============================================================================

func (h *TemplateHandler) HandleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	id := chi.URLParam(r, "id")

	// Verify access
	t, err := h.store.Get(ctx, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	if t.TenantID != tenantID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Soft delete (deprecated)
	if err := h.store.Delete(ctx, id, false); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// POST /api/semantic/templates/{id}/run - Execute Template
// ============================================================================

func (h *TemplateHandler) HandleRunTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	userID := r.Header.Get("X-User-ID")
	id := chi.URLParam(r, "id")

	// Load template
	tmpl, err := h.store.Get(ctx, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	if tmpl.TenantID != tenantID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Parse request
	var req TemplateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure region is present
	region := r.Header.Get("X-Tenant-Region")
	if region == "" {
		http.Error(w, "X-Tenant-Region header is required", http.StatusBadRequest)
		return
	}

	// Apply parameters to semantic query
	semQuery, err := ApplyTemplateParams(tmpl, req.Params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Parameter error: %v", err), http.StatusBadRequest)
		return
	}

	// Load bundle
	bundle, err := h.gateway.loadSemanticBundle(ctx, tenantID, tmpl.Datasource, region, tmpl.Version)
	if err != nil {
		http.Error(w, "Bundle not found", http.StatusNotFound)
		return
	}

	// Validate semantic query
	if err := h.gateway.server.ValidateSemanticQuery(bundle, semQuery); err != nil {
		http.Error(w, fmt.Sprintf("Query validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Generate SQL (with executor cache)
	start := time.Now()
	sql, err := h.gateway.callExecutorLLM(ctx, bundle, semQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("SQL generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute SQL (with results cache)
	rowsIface, err := h.gateway.executeSQL(ctx, sql)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert generic rows ([]interface{}) to []map[string]interface{}
	rows := make([]map[string]interface{}, 0, len(rowsIface))
	for _, rr := range rowsIface {
		if rm, ok := rr.(map[string]interface{}); ok {
			rows = append(rows, rm)
		} else {
			rows = append(rows, map[string]interface{}{"value": rr})
		}
	}

	elapsed := time.Since(start)

	// Record execution
	go h.recordTemplateExecution(context.Background(), id, userID, elapsed, len(rows))

	response := TemplateRunResponse{
		Datasource: tmpl.Datasource,
		Version:    tmpl.Version,
		SQL:        sql,
		Rows:       rows,
		Count:      len(rows),
		ExecutedAt: time.Now(),
		Duration:   elapsed.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// recordTemplateExecution logs template execution for metrics
func (h *TemplateHandler) recordTemplateExecution(ctx context.Context, templateID, userID string, duration time.Duration, rowCount int) {
	// This would insert into semantic_query_template_executions table
	// Implementation depends on your database schema
	log.Printf("Template executed: id=%s user=%s duration=%v rows=%d", templateID, userID, duration, rowCount)
}

// ============================================================================
// GET /api/semantic/templates/{id}/versions - List Template Versions
// ============================================================================

func (h *TemplateHandler) HandleListVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	id := chi.URLParam(r, "id")

	// Verify access
	t, err := h.store.Get(ctx, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	if t.TenantID != tenantID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	versions, err := h.store.ListVersions(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"versions": versions,
		"count":    len(versions),
	})
}

// ============================================================================
// GET /api/semantic/templates/{id}/diff - Diff Template Versions
// ============================================================================

type DiffResponse struct {
	FromVersion    int                    `json:"from_version"`
	ToVersion      int                    `json:"to_version"`
	QueryDiff      map[string]interface{} `json:"query_diff"`
	ParametersDiff map[string]interface{} `json:"parameters_diff"`
	NameChanged    bool                   `json:"name_changed"`
	OldName        string                 `json:"old_name,omitempty"`
	NewName        string                 `json:"new_name,omitempty"`
}

func (h *TemplateHandler) HandleDiffVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	id := chi.URLParam(r, "id")

	// Verify access
	t, err := h.store.Get(ctx, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	if t.TenantID != tenantID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Parse versions
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, _ := strconv.Atoi(fromStr)
	to, _ := strconv.Atoi(toStr)

	if from <= 0 || to <= 0 || from == to {
		http.Error(w, "Invalid from/to versions", http.StatusBadRequest)
		return
	}

	// Get versions
	v1, err := h.store.GetVersion(ctx, id, from)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	v2, err := h.store.GetVersion(ctx, id, to)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	// Compute diff
	diff := DiffResponse{
		FromVersion:    from,
		ToVersion:      to,
		QueryDiff:      diffJSON(toJSONMap(v1.SemanticQuery), toJSONMap(v2.SemanticQuery)),
		ParametersDiff: diffParametersList(v1.Parameters, v2.Parameters),
		NameChanged:    v1.Name != v2.Name,
		OldName:        v1.Name,
		NewName:        v2.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}

// Utility functions for diffing
func toJSONMap(v interface{}) map[string]interface{} {
	b, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}

func diffJSON(v1, v2 map[string]interface{}) map[string]interface{} {
	diff := make(map[string]interface{})
	diff["removed"] = make([]string, 0)
	diff["added"] = make([]string, 0)
	diff["changed"] = make(map[string]interface{})

	// Simplified diff - in production, use a proper diff library
	return diff
}

func diffParametersList(p1, p2 []TemplateParamDef) map[string]interface{} {
	diff := make(map[string]interface{})
	diff["removed"] = make([]string, 0)
	diff["added"] = make([]string, 0)
	diff["changed"] = make(map[string]interface{})

	return diff
}

// ============================================================================
// POST /api/semantic/templates/{id}/promote - Promote Version
// ============================================================================

func (h *TemplateHandler) HandlePromoteVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	id := chi.URLParam(r, "id")

	var req struct {
		VersionNumber int `json:"version"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Verify access
	t, err := h.store.Get(ctx, id)
	if err != nil {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	if t.TenantID != tenantID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	version, err := h.store.GetVersion(ctx, id, req.VersionNumber)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	// Update version to promoted
	now := time.Now()
	version.IsPromoted = true
	version.PromotedAt = &now

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// HandleGetVersion returns a specific version of the template
func (h *TemplateHandler) HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	vn := chi.URLParam(r, "versionNumber")

	versionNumber, err := strconv.Atoi(vn)
	if err != nil {
		http.Error(w, "invalid version number", http.StatusBadRequest)
		return
	}

	version, err := h.store.GetVersion(ctx, id, versionNumber)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// SetPermissions sets template permissions (stub)
func (h *TemplateHandler) SetPermissions(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// GetPermissions gets template permissions (stub)
func (h *TemplateHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// ============================================================================
// Template Handler Registration
// ============================================================================

// RegisterTemplateRoutes registers all template-related routes
func (s *Server) RegisterTemplateRoutes(router chi.Router) {
	store := NewTemplateStore(s.DB)
	gateway := NewLLMGateway(s)
	handler := NewTemplateHandler(store, gateway)

	router.Route("/templates", func(r chi.Router) {
		r.Post("/", handler.HandleCreateTemplate)
		r.Get("/", handler.HandleListTemplates)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", handler.HandleGetTemplate)
			r.Put("/", handler.HandleUpdateTemplate)
			r.Delete("/", handler.HandleDeleteTemplate)

			r.Post("/run", handler.HandleRunTemplate)
			r.Get("/versions", handler.HandleListVersions)
			r.Get("/diff", handler.HandleDiffVersions)
			r.Post("/promote", handler.HandlePromoteVersion)
		})
	})

	log.Printf("Template routes registered")
}
