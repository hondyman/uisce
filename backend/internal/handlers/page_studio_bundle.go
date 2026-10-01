package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Page bundles: a page with every fragment it depends on, as one portable
// document. Each fragment carries its content hash, so an import can tell
// "already have exactly this" from "have something different at the same
// version" (a conflict, never a silent overwrite) and a tampered or
// truncated bundle is refused.

const pageBundleFormat = "uisce.page-bundle/1"

type BundleFragment struct {
	Slug        string          `json:"slug"`
	Version     int             `json:"version"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	ContentHash string          `json:"contentHash"`
	Content     json.RawMessage `json:"content"`
}

type PageBundle struct {
	Format    string           `json:"format"`
	Page      json.RawMessage  `json:"page"` // the page as the page API shapes it, fragments named in app.fragments
	Fragments []BundleFragment `json:"fragments"`
}

type fragmentGetter func(slug string, version int) (*PageFragment, error)

// collectFragments walks references from the roots through each fragment's
// `uses` and returns the closure, ordered by slug then version.
func collectFragments(roots []fragmentRef, get fragmentGetter) ([]BundleFragment, error) {
	seen := map[fragmentRef]bool{}
	var out []BundleFragment
	var walk func(ref fragmentRef, depth int) error
	walk = func(ref fragmentRef, depth int) error {
		if seen[ref] {
			return nil
		}
		if depth > maxFragmentDepth {
			return fmt.Errorf("%q version %d is nested deeper than %d levels", ref.Fragment, ref.Version, maxFragmentDepth)
		}
		f, err := get(ref.Fragment, ref.Version)
		if err != nil {
			return err
		}
		if f == nil {
			return fmt.Errorf("the fragment %q version %d does not exist", ref.Fragment, ref.Version)
		}
		seen[ref] = true
		out = append(out, BundleFragment{Slug: f.Slug, Version: f.Version, Name: f.Name, Description: f.Description, ContentHash: f.ContentHash, Content: f.Content})
		var c fragmentContent
		_ = json.Unmarshal(f.Content, &c)
		for _, u := range c.Uses {
			if err := walk(u, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, r := range roots {
		if err := walk(r, 1); err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Slug != out[j].Slug {
			return out[i].Slug < out[j].Slug
		}
		return out[i].Version < out[j].Version
	})
	return out, nil
}

// pageFragmentRefs reads app.fragments from a page document.
func pageFragmentRefs(page json.RawMessage) []fragmentRef {
	var p struct {
		App struct {
			Fragments []fragmentRef `json:"fragments"`
		} `json:"app"`
	}
	_ = json.Unmarshal(page, &p)
	return p.App.Fragments
}

// verifyBundle refuses a bundle whose fragments do not match their hashes or
// that leaves a reference (from the page or a fragment) unresolved.
func verifyBundle(b PageBundle) error {
	if b.Format != pageBundleFormat {
		return fmt.Errorf("unsupported bundle format %q", b.Format)
	}
	have := map[fragmentRef]bool{}
	for _, f := range b.Fragments {
		canon, _, err := canonicalFragment(f.Content)
		if err != nil {
			return fmt.Errorf("fragment %q version %d: %w", f.Slug, f.Version, err)
		}
		if got := fragmentHash(canon); got != f.ContentHash {
			return fmt.Errorf("fragment %q version %d does not match its hash (content changed since export)", f.Slug, f.Version)
		}
		have[fragmentRef{f.Slug, f.Version}] = true
	}
	need := pageFragmentRefs(b.Page)
	for _, f := range b.Fragments {
		var c fragmentContent
		_ = json.Unmarshal(f.Content, &c)
		need = append(need, c.Uses...)
	}
	for _, r := range need {
		if !have[r] {
			return fmt.Errorf("the bundle is missing fragment %q version %d", r.Fragment, r.Version)
		}
	}
	return nil
}

// ImportPlan: what applying a bundle would do.
type ImportPlan struct {
	Create    []BundleFragment `json:"create"`
	Reuse     []fragmentRef    `json:"reuse"`
	Conflicts []string         `json:"conflicts"`
}

// planImport compares each fragment with what the tenant already has at that
// slug and version: absent = create, same hash = reuse, different = conflict.
func planImport(b PageBundle, existingHash func(slug string, version int) (string, bool, error)) (ImportPlan, error) {
	plan := ImportPlan{Create: []BundleFragment{}, Reuse: []fragmentRef{}, Conflicts: []string{}}
	for _, f := range b.Fragments {
		hash, found, err := existingHash(f.Slug, f.Version)
		if err != nil {
			return plan, err
		}
		switch {
		case !found:
			plan.Create = append(plan.Create, f)
		case hash == f.ContentHash:
			plan.Reuse = append(plan.Reuse, fragmentRef{f.Slug, f.Version})
		default:
			plan.Conflicts = append(plan.Conflicts, fmt.Sprintf("%q version %d already exists here with different content", f.Slug, f.Version))
		}
	}
	return plan, nil
}

func (h *PageStudioHandler) registerBundleRoutes(r chi.Router) {
	r.Get("/page-studio/pages/{id}/bundle", h.exportBundle)
	r.Post("/page-studio/bundles/import", h.importBundle)
}

func (h *PageStudioHandler) exportBundle(w http.ResponseWriter, r *http.Request) {
	if _, ok := mustTenantID(r); !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	page, err := h.getOne(r, "id = $1", id)
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
	if frags == nil {
		frags = []BundleFragment{}
	}
	writeJSON(w, http.StatusOK, PageBundle{Format: pageBundleFormat, Page: doc, Fragments: frags})
}

func (h *PageStudioHandler) existingFragmentHash(ctx context.Context, tenantID uuid.UUID) func(string, int) (string, bool, error) {
	return func(slug string, version int) (string, bool, error) {
		var hash string
		err := h.db.GetContext(ctx, &hash, `SELECT content_hash FROM page_fragments WHERE tenant_id = $1 AND slug = $2 AND version = $3`, tenantID, slug, version)
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return hash, err == nil, err
	}
}

// importBundle verifies a bundle and, unless ?dryRun=true, writes the
// fragments it plans to create (all or none). Any conflict refuses the whole
// import. It returns the plan and the page document; the client saves the
// page through the ordinary page API once its fragments exist.
func (h *PageStudioHandler) importBundle(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var b PageBundle
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := verifyBundle(b); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	plan, err := planImport(b, h.existingFragmentHash(r.Context(), tenantID))
	if err != nil {
		http.Error(w, "failed to plan import: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp := map[string]any{"plan": plan, "page": b.Page, "applied": false}
	if len(plan.Conflicts) > 0 {
		writeJSON(w, http.StatusConflict, resp)
		return
	}
	if r.URL.Query().Get("dryRun") == "true" {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to import: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() //nolint:errcheck
	for _, f := range plan.Create {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO page_fragments (tenant_id, slug, version, name, description, content, content_hash, is_core, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, false, $8)`, tenantID, f.Slug, f.Version, f.Name, f.Description, []byte(f.Content), f.ContentHash, userOf(r)); err != nil {
			http.Error(w, "failed to import fragment "+f.Slug+": "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to import: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp["applied"] = true
	writeJSON(w, http.StatusCreated, resp)
}
