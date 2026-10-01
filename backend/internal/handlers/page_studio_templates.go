package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Page templates (page_templates table): starting points for new pages, shown
// in a gallery. A template holds a page bundle - the page plus the fragments it
// depends on, each pinned by content hash - and is copied once into a new page.
// That is what separates it from a fragment, which pages reference live. It
// shares the bundle machinery (verify, plan, apply) and the immutable-version
// and hash identity with fragments, but is its own object type.

// templateContentKeys is what a template carries of a page: its content, never
// its identity (id, tenant, slug, version, status, timestamps, menu placements).
var templateContentKeys = []string{"description", "layout", "tabs", "components", "dataSources", "presentationEvents", "filterBar", "app"}

type PageTemplate struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	TenantID    uuid.UUID       `db:"tenant_id" json:"tenantId"`
	Slug        string          `db:"slug" json:"slug"`
	Version     int             `db:"version" json:"version"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	Category    string          `db:"category" json:"category"`
	Bundle      json.RawMessage `db:"bundle" json:"bundle,omitempty"`
	BundleHash  string          `db:"bundle_hash" json:"bundleHash"`
	IsCore      bool            `db:"is_core" json:"isCore"`
	CreatedAt   string          `db:"created_at" json:"createdAt"`
	CreatedBy   *string         `db:"created_by" json:"createdBy,omitempty"`
}

const templateColumns = `id, tenant_id, slug, version, name, description, category, bundle, bundle_hash, is_core, created_at::text AS created_at, created_by`
const templateSummaryColumns = `id, tenant_id, slug, version, name, description, category, '{}'::jsonb AS bundle, bundle_hash, is_core, created_at::text AS created_at, created_by`

// templateBundle strips a page down to its content and canonicalizes the
// bundle, so equal templates hash equal.
func templateBundle(pageDoc json.RawMessage, fragments []BundleFragment) (json.RawMessage, string, error) {
	var full map[string]json.RawMessage
	if err := json.Unmarshal(pageDoc, &full); err != nil {
		return nil, "", err
	}
	content := map[string]json.RawMessage{}
	for _, k := range templateContentKeys {
		if v, ok := full[k]; ok && string(v) != "null" {
			content[k] = v
		}
	}
	page, _ := json.Marshal(content)
	if fragments == nil {
		fragments = []BundleFragment{}
	}
	raw, err := json.Marshal(PageBundle{Format: pageBundleFormat, Page: page, Fragments: fragments})
	if err != nil {
		return nil, "", err
	}
	var generic any
	_ = json.Unmarshal(raw, &generic)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(generic); err != nil { // map keys encode sorted
		return nil, "", err
	}
	canon := bytes.TrimSpace(buf.Bytes())
	return canon, fragmentHash(canon), nil
}

// newPageFromTemplate: the template's page content under a new name and slug,
// as a draft. Identity never comes from the template.
func newPageFromTemplate(bundlePage json.RawMessage, name, slug string) (json.RawMessage, error) {
	var content map[string]json.RawMessage
	if err := json.Unmarshal(bundlePage, &content); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, k := range templateContentKeys {
		if v, ok := content[k]; ok {
			out[k] = v
		}
	}
	out["name"], out["slug"], out["status"] = name, slug, "draft"
	return json.Marshal(out)
}

func (h *PageStudioHandler) registerTemplateRoutes(r chi.Router) {
	r.Route("/page-studio/templates", func(r chi.Router) {
		r.Get("/", h.listTemplates)
		r.Post("/", h.publishTemplate)
		r.Get("/{slug}/versions/{version}", h.getTemplateVersion)
		r.Post("/{slug}/versions/{version}/instantiate", h.instantiateTemplate)
	})
}

func (h *PageStudioHandler) getTemplate(ctx context.Context, slug string, version int) (*PageTemplate, error) {
	var t PageTemplate
	err := h.db.GetContext(ctx, &t, `SELECT `+templateColumns+` FROM page_templates WHERE slug = $1 AND version = $2 ORDER BY is_core ASC LIMIT 1`, slug, version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

// listTemplates: the latest version of each template the caller can see, without
// its bundle; ?category= narrows the gallery.
func (h *PageStudioHandler) listTemplates(w http.ResponseWriter, r *http.Request) {
	if _, ok := mustTenantID(r); !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var out []PageTemplate
	if err := h.db.SelectContext(r.Context(), &out, `
		SELECT * FROM (SELECT DISTINCT ON (slug) `+templateSummaryColumns+` FROM page_templates ORDER BY slug, version DESC) t
		WHERE ($1 = '' OR category = $1) ORDER BY category, name`, r.URL.Query().Get("category")); err != nil {
		http.Error(w, "failed to list templates: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if out == nil {
		out = []PageTemplate{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *PageStudioHandler) getTemplateVersion(w http.ResponseWriter, r *http.Request) {
	if _, ok := mustTenantID(r); !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || version < 1 {
		http.Error(w, "invalid version", http.StatusBadRequest)
		return
	}
	t, err := h.getTemplate(r.Context(), chi.URLParam(r, "slug"), version)
	if err != nil {
		http.Error(w, "failed to load template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "template version not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

type publishTemplateRequest struct {
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	PageID      uuid.UUID `json:"pageId"`
	IsCore      *bool     `json:"isCore,omitempty"`
}

// publishTemplate saves a page as the next version of a template (1 for a new
// slug). The page's content and the fragments it depends on are captured at
// this moment; changing the page later does not change the template. The same
// content again is not a new version.
func (h *PageStudioHandler) publishTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req publishTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !fragmentSlugRE.MatchString(req.Slug) {
		http.Error(w, "slug must be lowercase letters, digits and hyphens, starting with a letter", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.PageID == uuid.Nil {
		http.Error(w, "name and pageId are required", http.StatusBadRequest)
		return
	}
	if req.Category == "" {
		req.Category = "general"
	}
	page, err := h.getOne(r, "id = $1", req.PageID)
	if err != nil || page == nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	doc, _ := json.Marshal(page)
	frags, err := collectFragments(pageFragmentRefs(doc), func(slug string, v int) (*PageFragment, error) { return h.getFragment(r.Context(), slug, v) })
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	bundle, hash, err := templateBundle(doc, frags)
	if err != nil {
		http.Error(w, "failed to build template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isCore := req.IsCore != nil && *req.IsCore && h.canEditCore(r, tenantID)

	var latest PageTemplate
	err = h.db.GetContext(r.Context(), &latest, `SELECT `+templateSummaryColumns+` FROM page_templates WHERE tenant_id = $1 AND slug = $2 ORDER BY version DESC LIMIT 1`, tenantID, req.Slug)
	if err == nil && latest.BundleHash == hash {
		writeJSON(w, http.StatusOK, latest)
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "failed to publish template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err == nil {
		if latest.IsCore && !h.canEditCore(r, tenantID) {
			http.Error(w, "core templates can only be changed by the gold-copy tenant admin", http.StatusForbidden)
			return
		}
		isCore = latest.IsCore
	}
	var t PageTemplate
	err = h.db.GetContext(r.Context(), &t, `
		INSERT INTO page_templates (tenant_id, slug, version, name, description, category, bundle, bundle_hash, is_core, created_by)
		VALUES ($1, $2, (SELECT COALESCE(MAX(version), 0) + 1 FROM page_templates WHERE tenant_id = $1 AND slug = $2), $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+templateSummaryColumns, tenantID, req.Slug, req.Name, req.Description, req.Category, []byte(bundle), hash, isCore, userOf(r))
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "another version was published at the same time; try again", http.StatusConflict)
			return
		}
		http.Error(w, "failed to publish template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

type instantiateRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// instantiateTemplate turns a template into a new draft page: it verifies the
// bundle, brings in the fragments it needs (a conflict refuses the whole
// thing; ?dryRun=true only plans), and returns the page document. The client
// saves the page through the ordinary page API, so every page rule applies.
func (h *PageStudioHandler) instantiateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || version < 1 {
		http.Error(w, "invalid version", http.StatusBadRequest)
		return
	}
	var req instantiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Slug == "" {
		http.Error(w, "name and slug are required", http.StatusBadRequest)
		return
	}
	t, err := h.getTemplate(r.Context(), chi.URLParam(r, "slug"), version)
	if err != nil {
		http.Error(w, "failed to load template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "template version not found", http.StatusNotFound)
		return
	}
	var b PageBundle
	if err := json.Unmarshal(t.Bundle, &b); err != nil {
		http.Error(w, "template bundle is unreadable", http.StatusInternalServerError)
		return
	}
	if err := verifyBundle(b); err != nil {
		http.Error(w, "template bundle is not valid: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	plan, err := planImport(b, h.existingFragmentHash(r.Context(), tenantID))
	if err != nil {
		http.Error(w, "failed to plan: "+err.Error(), http.StatusInternalServerError)
		return
	}
	page, err := newPageFromTemplate(b.Page, req.Name, req.Slug)
	if err != nil {
		http.Error(w, "template page is unreadable", http.StatusInternalServerError)
		return
	}
	resp := map[string]any{"plan": plan, "page": json.RawMessage(page), "applied": false}
	if len(plan.Conflicts) > 0 {
		writeJSON(w, http.StatusConflict, resp)
		return
	}
	if r.URL.Query().Get("dryRun") == "true" {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	if err := h.applyImportPlan(r.Context(), tenantID, userOf(r), plan); err != nil {
		http.Error(w, fmt.Sprintf("failed to bring in fragments: %v", err), http.StatusInternalServerError)
		return
	}
	resp["applied"] = true
	writeJSON(w, http.StatusCreated, resp)
}
