package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// PageRegistryEntry represents a declarative PageSpec blueprint stored in public.page_registry
type PageRegistryEntry struct {
	ID          string          `json:"id" db:"id"`
	TenantID    string          `json:"tenant_id" db:"tenant_id"`
	PageKey     string          `json:"page_key" db:"page_key"`
	Title       string          `json:"title" db:"title"`
	Description *string         `json:"description,omitempty" db:"description"`
	LayoutSpec  json.RawMessage `json:"layout_spec" db:"layout_spec"`
	IsGoldCopy  bool            `json:"is_gold_copy" db:"is_gold_copy"`
	IsActive    bool            `json:"is_active" db:"is_active"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

type PageDesignerLayoutHandler struct {
	db *sqlx.DB
}

func NewPageDesignerLayoutHandler(db *sqlx.DB) *PageDesignerLayoutHandler {
	return &PageDesignerLayoutHandler{db: db}
}

func (h *PageDesignerLayoutHandler) RegisterRoutes(r chi.Router) {
	r.Route("/page-designer", func(r chi.Router) {
		r.Get("/pages", h.ListPages)
		r.Post("/pages", h.SavePage)
		r.Get("/pages/{pageKey}", h.GetPage)
		r.Delete("/pages/{pageKey}", h.DeletePage)
	})
}

// GetPage resolves a page blueprint with 80/10/10 Gold Copy union fallback (Rule 1 & Rule 7)
func (h *PageDesignerLayoutHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	var tenantID string
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	pageKey := chi.URLParam(r, "pageKey")
	if pageKey == "" {
		http.Error(w, "pageKey is required", http.StatusBadRequest)
		return
	}

	var entry PageRegistryEntry
	// Query prioritizing tenant-specific blueprint, falling back to Gold Copy baseline
	query := `
		SELECT id, tenant_id, page_key, title, description, layout_spec, is_gold_copy, is_active, created_at, updated_at
		FROM public.page_registry
		WHERE page_key = $1 AND is_active = TRUE AND (tenant_id = $2 OR is_gold_copy = TRUE)
		ORDER BY CASE WHEN tenant_id = $2 THEN 0 ELSE 1 END, created_at DESC
		LIMIT 1;
	`
	err := h.db.GetContext(r.Context(), &entry, query, pageKey, tenantID)
	if err == sql.ErrNoRows {
		http.Error(w, fmt.Sprintf("Page not found: %s", pageKey), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch page: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entry)
}

// ListPages lists all active page blueprints accessible to the tenant (including Gold Copies)
func (h *PageDesignerLayoutHandler) ListPages(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	var tenantID string
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var pages []PageRegistryEntry
	query := `
		SELECT DISTINCT ON (page_key) id, tenant_id, page_key, title, description, layout_spec, is_gold_copy, is_active, created_at, updated_at
		FROM public.page_registry
		WHERE is_active = TRUE AND (tenant_id = $1 OR is_gold_copy = TRUE)
		ORDER BY page_key, CASE WHEN tenant_id = $1 THEN 0 ELSE 1 END, created_at DESC;
	`
	err := h.db.SelectContext(r.Context(), &pages, query, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pages: %v", err), http.StatusInternalServerError)
		return
	}
	if pages == nil {
		pages = []PageRegistryEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(pages)
}

// SavePage upserts a declarative page blueprint for the tenant (Rule 1 & Rule 7)
func (h *PageDesignerLayoutHandler) SavePage(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	var tenantID string
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	var req struct {
		PageKey     string          `json:"page_key"`
		Title       string          `json:"title"`
		Description *string         `json:"description,omitempty"`
		LayoutSpec  json.RawMessage `json:"layout_spec"`
		IsGoldCopy  bool            `json:"is_gold_copy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.PageKey == "" || req.Title == "" {
		http.Error(w, "page_key and title are required", http.StatusBadRequest)
		return
	}
	if len(req.LayoutSpec) == 0 {
		req.LayoutSpec = json.RawMessage("{}")
	}

	// Layer 3 Guardrail: Validate expression strings in layoutSpec against forbidden tokens
	layoutStr := strings.ToLower(string(req.LayoutSpec))
	forbiddenTokens := []string{"<script", "javascript:", "eval(", "exec(", "http://", "https://", "fetch(", "xhr"}
	for _, tok := range forbiddenTokens {
		if strings.Contains(layoutStr, tok) {
			http.Error(w, fmt.Sprintf("Forbidden token in presentation layout_spec: '%s' violates Presentation/Domain boundary (Rule 1 & Rule 6)", tok), http.StatusBadRequest)
			return
		}
	}

	newID := uuid.New().String()
	query := `
		INSERT INTO public.page_registry (id, tenant_id, page_key, title, description, layout_spec, is_gold_copy, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, NOW(), NOW())
		ON CONFLICT (tenant_id, page_key)
		DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			layout_spec = EXCLUDED.layout_spec,
			is_gold_copy = EXCLUDED.is_gold_copy,
			is_active = TRUE,
			updated_at = NOW()
		RETURNING id, tenant_id, page_key, title, description, layout_spec, is_gold_copy, is_active, created_at, updated_at;
	`

	var saved PageRegistryEntry
	err := h.db.GetContext(r.Context(), &saved, query, newID, tenantID, req.PageKey, req.Title, req.Description, req.LayoutSpec, req.IsGoldCopy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save page blueprint: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(saved)
}

// DeletePage marks a page blueprint as inactive
func (h *PageDesignerLayoutHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	var tenantID string
	if claims != nil {
		tenantID = claims.TenantID
	}
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}

	pageKey := chi.URLParam(r, "pageKey")
	result, err := h.db.ExecContext(r.Context(),
		`UPDATE public.page_registry SET is_active = FALSE, updated_at = NOW() WHERE page_key = $1 AND tenant_id = $2`,
		pageKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete page: %v", err), http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
