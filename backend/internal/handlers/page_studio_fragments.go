package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Page fragments (page_fragments table): reusable pieces of a page that pages
// and other fragments reference by slug and exact version. A version is
// immutable; publishing changes creates the next one. Nesting is capped at
// maxFragmentDepth (a page uses a fragment, which may use one more level).

const maxFragmentDepth = 2

var fragmentSlugRE = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

type fragmentRef struct {
	Fragment string `json:"fragment"`
	Version  int    `json:"version"`
}

// fragmentContent is what a fragment holds; Uses names the fragments it is built from.
type fragmentContent struct {
	Root       string          `json:"root,omitempty"`
	Components json.RawMessage `json:"components"`
	Nodes      json.RawMessage `json:"nodes"`
	Variables  json.RawMessage `json:"variables"`
	Queries    json.RawMessage `json:"queries"`
	Uses       []fragmentRef   `json:"uses,omitempty"`
}

type PageFragment struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	TenantID    uuid.UUID       `db:"tenant_id" json:"tenantId"`
	Slug        string          `db:"slug" json:"slug"`
	Version     int             `db:"version" json:"version"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	Content     json.RawMessage `db:"content" json:"content"`
	ContentHash string          `db:"content_hash" json:"contentHash"`
	IsCore      bool            `db:"is_core" json:"isCore"`
	CreatedAt   string          `db:"created_at" json:"createdAt"`
	CreatedBy   *string         `db:"created_by" json:"createdBy,omitempty"`
}

const fragmentColumns = `id, tenant_id, slug, version, name, description, content, content_hash, is_core, created_at::text AS created_at, created_by`

// canonicalFragment normalizes content (missing lists become empty, keys are
// sorted by encoding through a generic value) so equal content hashes equal.
func canonicalFragment(raw json.RawMessage) (json.RawMessage, fragmentContent, error) {
	var c fragmentContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, c, fmt.Errorf("content is not valid JSON: %w", err)
	}
	for _, f := range []*json.RawMessage{&c.Components, &c.Nodes} {
		if len(*f) == 0 || string(*f) == "null" {
			*f = json.RawMessage(`{}`)
		}
	}
	for _, f := range []*json.RawMessage{&c.Variables, &c.Queries} {
		if len(*f) == 0 || string(*f) == "null" {
			*f = json.RawMessage(`[]`)
		}
	}
	var generic any
	whole, _ := json.Marshal(c)
	_ = json.Unmarshal(whole, &generic)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(generic); err != nil { // map keys encode sorted
		return nil, c, err
	}
	return bytes.TrimSpace(buf.Bytes()), c, nil
}

func fragmentHash(canonical []byte) string {
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

func (h *PageStudioHandler) registerFragmentRoutes(r chi.Router) {
	r.Route("/page-studio/fragments", func(r chi.Router) {
		r.Get("/", h.listFragments)
		r.Post("/", h.publishFragment)
		r.Get("/{slug}/versions/{version}", h.getFragmentVersion)
		r.Get("/{slug}/usage", h.fragmentUsage)
	})
}

func (h *PageStudioHandler) getFragment(ctx context.Context, slug string, version int) (*PageFragment, error) {
	var f PageFragment
	err := h.db.GetContext(ctx, &f, `SELECT `+fragmentColumns+` FROM page_fragments WHERE slug = $1 AND version = $2 ORDER BY (is_core) ASC LIMIT 1`, slug, version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &f, err
}

// checkFragmentUses: every referenced version exists, and nesting stays within the cap.
func (h *PageStudioHandler) checkFragmentUses(ctx context.Context, slug string, uses []fragmentRef) error {
	for _, u := range uses {
		if u.Fragment == slug {
			return fmt.Errorf("a fragment cannot use itself (%q)", slug)
		}
		got, err := h.getFragment(ctx, u.Fragment, u.Version)
		if err != nil {
			return err
		}
		if got == nil {
			return fmt.Errorf("the fragment %q version %d does not exist", u.Fragment, u.Version)
		}
		var inner fragmentContent
		_ = json.Unmarshal(got.Content, &inner)
		if len(inner.Uses) > 0 {
			return fmt.Errorf("%q uses other fragments itself; fragments nest at most %d levels", u.Fragment, maxFragmentDepth)
		}
	}
	return nil
}

func (h *PageStudioHandler) listFragments(w http.ResponseWriter, r *http.Request) {
	if _, ok := mustTenantID(r); !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var out []PageFragment
	// The latest version of each slug the caller can see.
	if err := h.db.SelectContext(r.Context(), &out, `
		SELECT DISTINCT ON (slug) `+fragmentColumns+` FROM page_fragments ORDER BY slug, version DESC`); err != nil {
		http.Error(w, "failed to list fragments: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if out == nil {
		out = []PageFragment{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *PageStudioHandler) getFragmentVersion(w http.ResponseWriter, r *http.Request) {
	if _, ok := mustTenantID(r); !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || version < 1 {
		http.Error(w, "invalid version", http.StatusBadRequest)
		return
	}
	f, err := h.getFragment(r.Context(), chi.URLParam(r, "slug"), version)
	if err != nil {
		http.Error(w, "failed to load fragment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if f == nil {
		http.Error(w, "fragment version not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

type publishFragmentRequest struct {
	Slug        string          `json:"slug"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Content     json.RawMessage `json:"content"`
	IsCore      *bool           `json:"isCore,omitempty"`
}

// publishFragment writes the next version of a fragment (1 for a new slug).
// Identical content to the latest version is not a new version.
func (h *PageStudioHandler) publishFragment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req publishFragmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !fragmentSlugRE.MatchString(req.Slug) {
		http.Error(w, "slug must be lowercase letters, digits and hyphens, starting with a letter", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	canonical, content, err := canonicalFragment(req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.checkFragmentUses(r.Context(), req.Slug, content.Uses); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	isCore := (req.IsCore != nil && *req.IsCore) && h.canEditCore(r, tenantID)
	hash := fragmentHash(canonical)

	var latest PageFragment
	err = h.db.GetContext(r.Context(), &latest, `SELECT `+fragmentColumns+` FROM page_fragments WHERE tenant_id = $1 AND slug = $2 ORDER BY version DESC LIMIT 1`, tenantID, req.Slug)
	if err == nil && latest.ContentHash == hash {
		writeJSON(w, http.StatusOK, latest)
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "failed to publish fragment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err == nil && latest.IsCore && !h.canEditCore(r, tenantID) {
		http.Error(w, "core fragments can only be changed by the gold-copy tenant admin", http.StatusForbidden)
		return
	}
	if err == nil {
		isCore = latest.IsCore
	}
	var f PageFragment
	err = h.db.GetContext(r.Context(), &f, `
		INSERT INTO page_fragments (tenant_id, slug, version, name, description, content, content_hash, is_core, created_by)
		VALUES ($1, $2, (SELECT COALESCE(MAX(version), 0) + 1 FROM page_fragments WHERE tenant_id = $1 AND slug = $2), $3, $4, $5, $6, $7, $8)
		RETURNING `+fragmentColumns, tenantID, req.Slug, req.Name, req.Description, canonical, hash, isCore, userOf(r))
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "another version was published at the same time; try again", http.StatusConflict)
			return
		}
		http.Error(w, "failed to publish fragment: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// fragmentUse is one place that references a fragment.
type fragmentUse struct {
	Kind    string `db:"kind" json:"kind"` // "page" or "fragment"
	Slug    string `db:"slug" json:"slug"`
	Name    string `db:"name" json:"name"`
	Version int    `db:"version" json:"version"` // the fragment version it pins
}

// fragmentUsage finds what references a fragment, at any version, among the
// pages (app.fragments) and fragments (content.uses) the caller can see. It
// is what makes "can I change or switch off this fragment?" answerable.
func (h *PageStudioHandler) fragmentUsage(w http.ResponseWriter, r *http.Request) {
	if _, ok := mustTenantID(r); !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	uses, err := h.usesOfFragment(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		http.Error(w, "failed to scan usage: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, uses)
}

func (h *PageStudioHandler) usesOfFragment(ctx context.Context, slug string) ([]fragmentUse, error) {
	var uses []fragmentUse
	err := h.db.SelectContext(ctx, &uses, `
		SELECT 'page' AS kind, p.slug, p.name, (ref->>'version')::int AS version
		  FROM page_definitions p, jsonb_array_elements(COALESCE(p.app_model->'fragments', '[]'::jsonb)) ref
		 WHERE ref->>'fragment' = $1
		UNION ALL
		SELECT 'fragment', f.slug, f.name, (ref->>'version')::int
		  FROM page_fragments f, jsonb_array_elements(COALESCE(f.content->'uses', '[]'::jsonb)) ref
		 WHERE ref->>'fragment' = $1
		ORDER BY 1, 2`, slug)
	if uses == nil {
		uses = []fragmentUse{}
	}
	return uses, err
}
