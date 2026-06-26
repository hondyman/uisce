package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// PageLayout represents a saved page layout configuration
type PageLayout struct {
	ID             string          `json:"id" db:"id"`
	TenantID       string          `json:"tenant_id" db:"tenant_id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description,omitempty" db:"description"`
	PrimaryBO      string          `json:"primary_bo" db:"primary_bo"`
	LayoutType     string          `json:"layout_type" db:"layout_type"`
	LayoutJSON     json.RawMessage `json:"layout_json" db:"layout_json"`
	PipelineID     *string         `json:"pipeline_id,omitempty" db:"pipeline_id"`
	IsActive       bool            `json:"is_active" db:"is_active"`
	CreatedBy      string          `json:"created_by,omitempty" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	LastModifiedAt time.Time       `json:"last_modified_at" db:"last_modified_at"`
}

// PageLayoutHandler handles page layout CRUD operations
type PageLayoutHandler struct {
	db *sqlx.DB
}

// NewPageLayoutHandler creates a new handler
func NewPageLayoutHandler(db *sqlx.DB) *PageLayoutHandler {
	return &PageLayoutHandler{db: db}
}

// RegisterRoutes registers the page layout routes (compatible with existing frontend)
func (h *PageLayoutHandler) RegisterRoutes(r chi.Router) {
	// Compatible /api/layouts routes
	r.Get("/api/layouts", h.ListPageLayouts)
	r.Post("/api/layouts", h.CreatePageLayout)
	r.Get("/api/layouts/{id}", h.GetPageLayout)
	r.Put("/api/layouts/{id}", h.UpdatePageLayout)
	r.Delete("/api/layouts/{id}", h.DeletePageLayout)

	// v1 namespace
	r.Get("/api/v1/page-layouts", h.ListPageLayouts)
	r.Post("/api/v1/page-layouts", h.CreatePageLayout)
	r.Get("/api/v1/page-layouts/{id}", h.GetPageLayout)
	r.Put("/api/v1/page-layouts/{id}", h.UpdatePageLayout)
	r.Delete("/api/v1/page-layouts/{id}", h.DeletePageLayout)
}

// ListPageLayouts returns all page layouts for a tenant
func (h *PageLayoutHandler) ListPageLayouts(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001" // Default tenant
	}

	boFilter := r.URL.Query().Get("primary_bo")

	var layouts []PageLayout
	var err error

	if boFilter != "" {
		err = h.db.SelectContext(r.Context(), &layouts,
			`SELECT id, tenant_id, name, description, primary_bo, layout_type, layout_json, 
			        pipeline_id, is_active, created_by, created_at, last_modified_at
			 FROM page_layouts 
			 WHERE tenant_id = $1 AND primary_bo = $2 AND is_active = true
			 ORDER BY name`, tenantID, boFilter)
	} else {
		err = h.db.SelectContext(r.Context(), &layouts,
			`SELECT id, tenant_id, name, description, primary_bo, layout_type, layout_json, 
			        pipeline_id, is_active, created_by, created_at, last_modified_at
			 FROM page_layouts 
			 WHERE tenant_id = $1 AND is_active = true
			 ORDER BY name`, tenantID)
	}

	if err != nil {
		http.Error(w, "Failed to list page layouts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if layouts == nil {
		layouts = []PageLayout{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(layouts)
}

// CreatePageLayout creates a new page layout
func (h *PageLayoutHandler) CreatePageLayout(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "system"
	}

	var layout PageLayout
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	layout.ID = uuid.New().String()
	layout.TenantID = tenantID
	layout.CreatedBy = userID
	layout.IsActive = true
	if layout.LayoutType == "" {
		layout.LayoutType = "form"
	}

	query := `
		INSERT INTO page_layouts (id, tenant_id, name, description, primary_bo, layout_type, layout_json, pipeline_id, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, last_modified_at
	`

	err := h.db.QueryRowContext(r.Context(), query,
		layout.ID, layout.TenantID, layout.Name, layout.Description,
		layout.PrimaryBO, layout.LayoutType, layout.LayoutJSON,
		layout.PipelineID, layout.IsActive, layout.CreatedBy,
	).Scan(&layout.CreatedAt, &layout.LastModifiedAt)

	if err != nil {
		http.Error(w, "Failed to create page layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(layout)
}

// GetPageLayout returns a single page layout by ID
func (h *PageLayoutHandler) GetPageLayout(w http.ResponseWriter, r *http.Request) {
	layoutID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var layout PageLayout
	err := h.db.GetContext(r.Context(), &layout,
		`SELECT id, tenant_id, name, description, primary_bo, layout_type, layout_json, 
		        pipeline_id, is_active, created_by, created_at, last_modified_at
		 FROM page_layouts 
		 WHERE id = $1 AND tenant_id = $2`, layoutID, tenantID)

	if err == sql.ErrNoRows {
		http.Error(w, "Page layout not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to get page layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(layout)
}

// UpdatePageLayout updates an existing page layout
func (h *PageLayoutHandler) UpdatePageLayout(w http.ResponseWriter, r *http.Request) {
	layoutID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var layout PageLayout
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE page_layouts 
		SET name = $1, description = $2, primary_bo = $3, layout_type = $4, 
		    layout_json = $5, pipeline_id = $6, last_modified_at = NOW()
		WHERE id = $7 AND tenant_id = $8
		RETURNING id, tenant_id, name, description, primary_bo, layout_type, layout_json, 
		          pipeline_id, is_active, created_by, created_at, last_modified_at
	`

	var updated PageLayout
	err := h.db.QueryRowContext(r.Context(), query,
		layout.Name, layout.Description, layout.PrimaryBO, layout.LayoutType,
		layout.LayoutJSON, layout.PipelineID, layoutID, tenantID,
	).Scan(&updated.ID, &updated.TenantID, &updated.Name, &updated.Description,
		&updated.PrimaryBO, &updated.LayoutType, &updated.LayoutJSON,
		&updated.PipelineID, &updated.IsActive, &updated.CreatedBy,
		&updated.CreatedAt, &updated.LastModifiedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Page layout not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to update page layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// DeletePageLayout soft-deletes a page layout
func (h *PageLayoutHandler) DeletePageLayout(w http.ResponseWriter, r *http.Request) {
	layoutID := chi.URLParam(r, "id")

	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	result, err := h.db.ExecContext(r.Context(),
		`UPDATE page_layouts SET is_active = false, last_modified_at = NOW() 
		 WHERE id = $1 AND tenant_id = $2`, layoutID, tenantID)

	if err != nil {
		http.Error(w, "Failed to delete page layout: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Page layout not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
