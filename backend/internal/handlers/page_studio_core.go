package handlers

// Tenant lifecycle for core (gold-copy) pages. A tenant never writes a core
// page; for each one it chooses (public.core_object_adoption, object_type
// 'page'):
//
//  1. inactive  - switched off in this tenant (active = false)
//  2. vanilla   - used as the gold copy ships it (no row)
//  3. extended  - customized; customizations are diffed against the core
//                 version they were made on and carried forward (or
//                 dropped, one by one) when the core moves on
//  4. cloned    - an independent tenant page that replaces the core page
//                 at the same slug; no upgrade path
//
// The diff/merge is generic (internal/corecustom); this file is the page
// half: what a page's content is, how its changes group into the units a
// person decides about, and the endpoints.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/corecustom"
	"github.com/hondyman/uisce/backend/internal/security"
)

const corePageObjectType = "page"

// PageCustomization is how the calling tenant uses a core page.
type PageCustomization struct {
	Mode   string `json:"mode"` // vanilla | extended | cloned
	Active bool   `json:"active"`
	// CoreVersion is the gold copy's current version; BaseVersion the one
	// an extension was made on (or a clone taken from).
	CoreVersion      int        `json:"coreVersion"`
	BaseVersion      int        `json:"baseVersion,omitempty"`
	UpgradeAvailable bool       `json:"upgradeAvailable"`
	ClonePageID      *uuid.UUID `json:"clonePageId,omitempty"`
}

// PageCloneSource marks a tenant page as a clone of a core page.
type PageCloneSource struct {
	PageID  uuid.UUID `json:"pageId"`
	Name    string    `json:"name"`
	Version int       `json:"version"`
}

// pageContent is the part of a page a tenant can customize - what gets
// diffed. Identity (id, slug), lifecycle (status, version) and ownership
// are not content.
type pageContent struct {
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Layout             json.RawMessage `json:"layout"`
	Tabs               json.RawMessage `json:"tabs"`
	Components         json.RawMessage `json:"components"`
	DataSources        json.RawMessage `json:"dataSources"`
	PresentationEvents json.RawMessage `json:"presentationEvents"`
	FilterBar          json.RawMessage `json:"filterBar"`
	App                json.RawMessage `json:"app"`
}

func contentOf(p *PageStudioPage) pageContent {
	c := pageContent{
		Name: p.Name, Description: p.Description, Layout: p.Layout, Tabs: p.Tabs, Components: p.Components,
		DataSources: p.DataSources, PresentationEvents: p.PresentationEvents, FilterBar: p.FilterBar,
	}
	if p.App != nil {
		c.App = *p.App
	}
	return c.normalized()
}

func contentOfRequest(req *pageStudioUpsertRequest) pageContent {
	normalizeUpsertDefaults(req)
	return pageContent{
		Name: req.Name, Description: req.Description, Layout: req.Layout, Tabs: req.Tabs, Components: req.Components,
		DataSources: req.DataSources, PresentationEvents: req.PresentationEvents, FilterBar: req.FilterBar, App: req.App,
	}.normalized()
}

// normalized gives every field a JSON value, so an absent field and its
// empty default never show up as a customization.
func (c pageContent) normalized() pageContent {
	def := func(raw json.RawMessage, empty string) json.RawMessage {
		if len(raw) == 0 {
			return json.RawMessage(empty)
		}
		return raw
	}
	c.Layout = def(c.Layout, `[]`)
	c.Tabs = def(c.Tabs, `[]`)
	c.Components = def(c.Components, `{}`)
	c.DataSources = def(c.DataSources, `[]`)
	c.PresentationEvents = def(c.PresentationEvents, `[]`)
	c.FilterBar = def(c.FilterBar, string(emptyPageLayoutJSON))
	c.App = def(c.App, `null`)
	return c
}

func (c pageContent) applyTo(p *PageStudioPage) {
	p.Name, p.Description = c.Name, c.Description
	p.Layout, p.Tabs, p.Components, p.DataSources = c.Layout, c.Tabs, c.Components, c.DataSources
	p.PresentationEvents, p.FilterBar = c.PresentationEvents, c.FilterBar
	if len(c.App) == 0 || string(c.App) == "null" {
		p.App = nil
	} else {
		app := c.App
		p.App = &app
	}
}

func (c pageContent) doc() any {
	b, _ := json.Marshal(c)
	v, _ := corecustom.Decode(b)
	return v
}

func contentFromDoc(v any) (pageContent, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return pageContent{}, err
	}
	var c pageContent
	if err := json.Unmarshal(b, &c); err != nil {
		return pageContent{}, err
	}
	return c.normalized(), nil
}

func contentFromJSON(raw []byte) (pageContent, error) {
	var c pageContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return pageContent{}, err
	}
	return c.normalized(), nil
}

// pageAdoption is one core_object_adoption row for a page.
type pageAdoption struct {
	CoreObjectID  uuid.UUID     `db:"core_object_id"`
	Active        bool          `db:"active"`
	Mode          string        `db:"mode"`
	BaseVersion   sql.NullInt64 `db:"base_version"`
	BaseSnapshot  []byte        `db:"base_snapshot"`
	Extension     []byte        `db:"extension"`
	CloneObjectID uuid.NullUUID `db:"clone_object_id"`
}

const adoptionColumns = `core_object_id, active, mode, base_version, base_snapshot, extension, clone_object_id`

func (h *PageStudioHandler) adoptions(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]pageAdoption, error) {
	var rows []pageAdoption
	if err := h.db.SelectContext(ctx, &rows, `SELECT `+adoptionColumns+` FROM core_object_adoption WHERE tenant_id = $1 AND object_type = $2`, tenantID, corePageObjectType); err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]pageAdoption, len(rows))
	for _, a := range rows {
		out[a.CoreObjectID] = a
	}
	return out, nil
}

func (h *PageStudioHandler) adoption(ctx context.Context, tenantID, coreID uuid.UUID) (*pageAdoption, error) {
	var a pageAdoption
	err := h.db.GetContext(ctx, &a, `SELECT `+adoptionColumns+` FROM core_object_adoption WHERE tenant_id = $1 AND object_type = $2 AND core_object_id = $3`, tenantID, corePageObjectType, coreID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// presentCore turns the gold copy's page into what this tenant sees: the
// extension's content when extended, plus how the tenant uses it. Tenants
// never edit a core page in place; canCustomize says whether this caller
// may extend / clone / switch it off.
func presentCore(p *PageStudioPage, a *pageAdoption, canCustomize bool) {
	c := &PageCustomization{Mode: "vanilla", Active: true, CoreVersion: p.Version}
	if a != nil {
		c.Mode, c.Active = a.Mode, a.Active
		if a.BaseVersion.Valid {
			c.BaseVersion = int(a.BaseVersion.Int64)
		}
		switch a.Mode {
		case "extended":
			if ext, err := contentFromJSON(a.Extension); err == nil {
				ext.applyTo(p)
			}
			c.UpgradeAvailable = c.BaseVersion < c.CoreVersion
		case "cloned":
			if a.CloneObjectID.Valid {
				id := a.CloneObjectID.UUID
				c.ClonePageID = &id
			}
		}
	}
	p.Customization = c
	p.Editable = false
	p.CanCustomize = canCustomize
}

// canCustomize: a tenant admin (or a global admin working in the tenant)
// in a tenant that is not the gold copy. The gold copy edits its core
// pages directly (canEditCore) instead.
func (h *PageStudioHandler) canCustomize(r *http.Request, tenantID uuid.UUID) bool {
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok {
		return false
	}
	allowed := auth.IsGlobalAdmin
	for _, role := range auth.Roles {
		if role == "tenant_admin" || role == "admin" {
			allowed = true
		}
	}
	if !allowed {
		return false
	}
	gold := h.goldCopyID(r.Context())
	return gold != uuid.Nil && gold != tenantID
}

// corePageRequest is the common preamble of every lifecycle endpoint: the
// caller may customize, and {id} is a core page of the gold copy.
func (h *PageStudioHandler) corePageRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, *PageStudioPage, *pageAdoption, bool) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return uuid.Nil, nil, nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return uuid.Nil, nil, nil, false
	}
	if !h.canCustomize(r, tenantID) {
		http.Error(w, "customizing core pages needs a tenant admin outside the gold copy", http.StatusForbidden)
		return uuid.Nil, nil, nil, false
	}
	page, err := h.getOne(r, "id = $1 AND is_core = true AND tenant_id = $2", id, h.goldCopyID(r.Context()))
	if err == sql.ErrNoRows {
		http.Error(w, "core page not found", http.StatusNotFound)
		return uuid.Nil, nil, nil, false
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return uuid.Nil, nil, nil, false
	}
	a, err := h.adoption(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, "failed to load customization: "+err.Error(), http.StatusInternalServerError)
		return uuid.Nil, nil, nil, false
	}
	return tenantID, page, a, true
}

func userOf(r *http.Request) string {
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok {
		return auth.UserID
	}
	return ""
}

// saveExtension stores the tenant's customized copy of a core page. The
// first save pins the base (the core as it is now); later saves keep that
// base so an upgrade can still tell the tenant's changes from the core's.
func (h *PageStudioHandler) saveExtension(w http.ResponseWriter, r *http.Request) {
	tenantID, core, a, ok := h.corePageRequest(w, r)
	if !ok {
		return
	}
	if a != nil && a.Mode == "cloned" {
		http.Error(w, "this core page is cloned in your environment; edit the clone, or revert to core first", http.StatusConflict)
		return
	}
	var req pageStudioUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	ext, err := json.Marshal(contentOfRequest(&req))
	if err != nil {
		http.Error(w, "invalid page content", http.StatusBadRequest)
		return
	}
	base, err := json.Marshal(contentOf(core))
	if err != nil {
		http.Error(w, "failed to snapshot core page", http.StatusInternalServerError)
		return
	}
	_, err = h.db.ExecContext(r.Context(), `
		INSERT INTO core_object_adoption (tenant_id, object_type, core_object_id, mode, base_version, base_snapshot, extension, updated_at, updated_by)
		VALUES ($1, $2, $3, 'extended', $4, $5, $6, NOW(), $7)
		ON CONFLICT (tenant_id, object_type, core_object_id) DO UPDATE SET
			mode = 'extended',
			base_version = CASE WHEN core_object_adoption.mode = 'extended' THEN core_object_adoption.base_version ELSE EXCLUDED.base_version END,
			base_snapshot = CASE WHEN core_object_adoption.mode = 'extended' THEN core_object_adoption.base_snapshot ELSE EXCLUDED.base_snapshot END,
			extension = EXCLUDED.extension,
			updated_at = NOW(), updated_by = EXCLUDED.updated_by
	`, tenantID, corePageObjectType, core.ID, core.Version, base, ext, userOf(r))
	if err != nil {
		http.Error(w, "failed to save customization: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.respondCore(w, r, tenantID, core)
}

// setActivation switches a core page on or off for this tenant.
func (h *PageStudioHandler) setActivation(w http.ResponseWriter, r *http.Request) {
	tenantID, core, _, ok := h.corePageRequest(w, r)
	if !ok {
		return
	}
	var req struct {
		Active *bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Active == nil {
		http.Error(w, "active (true or false) is required", http.StatusBadRequest)
		return
	}
	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO core_object_adoption (tenant_id, object_type, core_object_id, active, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, NOW(), $5)
		ON CONFLICT (tenant_id, object_type, core_object_id) DO UPDATE SET
			active = EXCLUDED.active, updated_at = NOW(), updated_by = EXCLUDED.updated_by
	`, tenantID, corePageObjectType, core.ID, *req.Active, userOf(r))
	if err == nil {
		// Active and vanilla is the default: keep no row for it.
		_, err = h.db.ExecContext(r.Context(), `
			DELETE FROM core_object_adoption
			WHERE tenant_id = $1 AND object_type = $2 AND core_object_id = $3 AND active AND mode = 'vanilla'
		`, tenantID, corePageObjectType, core.ID)
	}
	if err != nil {
		http.Error(w, "failed to change activation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.respondCore(w, r, tenantID, core)
}

// cloneCore makes an independent tenant copy of a core page (its current
// extension, if any) at the same slug, so it replaces the core page in
// this tenant. A clone gets no upgrades.
func (h *PageStudioHandler) cloneCore(w http.ResponseWriter, r *http.Request) {
	tenantID, core, a, ok := h.corePageRequest(w, r)
	if !ok {
		return
	}
	if a != nil && a.Mode == "cloned" {
		http.Error(w, "this core page is already cloned in your environment", http.StatusConflict)
		return
	}
	presentCore(core, a, true)
	c := contentOf(core)

	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	cloneID := uuid.New()
	var clone PageStudioPage
	err = tx.GetContext(r.Context(), &clone, `
		INSERT INTO page_definitions (id, tenant_id, name, slug, description, layout, tabs, components, data_sources, presentation_events, filter_bar, version, is_core, status, app_model)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 1, false, $12, $13)
		RETURNING `+pageColumns,
		cloneID, tenantID, c.Name, core.Slug, c.Description, c.Layout, c.Tabs, c.Components, c.DataSources,
		c.PresentationEvents, c.FilterBar, core.Status, nullableJSON(c.App))
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "your environment already has a page at /"+core.Slug+"; rename or delete it first", http.StatusConflict)
			return
		}
		http.Error(w, "failed to clone page: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = tx.ExecContext(r.Context(), `
		INSERT INTO core_object_adoption (tenant_id, object_type, core_object_id, active, mode, base_version, clone_object_id, updated_at, updated_by)
		VALUES ($1, $2, $3, true, 'cloned', $4, $5, NOW(), $6)
		ON CONFLICT (tenant_id, object_type, core_object_id) DO UPDATE SET
			mode = 'cloned', base_version = EXCLUDED.base_version, base_snapshot = NULL, extension = NULL,
			clone_object_id = EXCLUDED.clone_object_id, updated_at = NOW(), updated_by = EXCLUDED.updated_by
	`, tenantID, corePageObjectType, core.ID, core.Customization.CoreVersion, cloneID, userOf(r))
	if err != nil {
		http.Error(w, "failed to record clone: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit: "+err.Error(), http.StatusInternalServerError)
		return
	}
	clone.Editable = true
	clone.ClonedFrom = &PageCloneSource{PageID: core.ID, Name: core.Name, Version: core.Customization.CoreVersion}
	writeJSON(w, http.StatusCreated, clone)
}

// revertToCore drops the tenant's extension or clone (the clone page is
// deleted) and puts the core page back as the gold copy ships it, active.
func (h *PageStudioHandler) revertToCore(w http.ResponseWriter, r *http.Request) {
	tenantID, core, a, ok := h.corePageRequest(w, r)
	if !ok {
		return
	}
	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if a != nil && a.CloneObjectID.Valid {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM page_definitions WHERE id = $1 AND tenant_id = $2 AND is_core = false`, a.CloneObjectID.UUID, tenantID); err != nil {
			http.Error(w, "failed to delete clone: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM core_object_adoption WHERE tenant_id = $1 AND object_type = $2 AND core_object_id = $3`, tenantID, corePageObjectType, core.ID); err != nil {
		http.Error(w, "failed to revert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.respondCore(w, r, tenantID, core)
}

// PageComparison is compare's response: the tenant's customizations and
// the core's changes since the tenant's base, grouped by element.
type PageComparison struct {
	BaseVersion      int                `json:"baseVersion"`
	CoreVersion      int                `json:"coreVersion"`
	UpgradeAvailable bool               `json:"upgradeAvailable"`
	Customizations   []corecustom.Group `json:"customizations"`
	CoreUpdates      []corecustom.Group `json:"coreUpdates"`
}

// extensionDocs loads base, extension and current core as documents.
func extensionDocs(core *PageStudioPage, a *pageAdoption) (base, ext, cur any, err error) {
	baseC, err := contentFromJSON(a.BaseSnapshot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stored base is not a page: %w", err)
	}
	extC, err := contentFromJSON(a.Extension)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("stored extension is not a page: %w", err)
	}
	return baseC.doc(), extC.doc(), contentOf(core).doc(), nil
}

func (h *PageStudioHandler) compare(w http.ResponseWriter, r *http.Request) {
	_, core, a, ok := h.corePageRequest(w, r)
	if !ok {
		return
	}
	if a == nil || a.Mode != "extended" {
		writeJSON(w, http.StatusOK, PageComparison{BaseVersion: core.Version, CoreVersion: core.Version,
			Customizations: []corecustom.Group{}, CoreUpdates: []corecustom.Group{}})
		return
	}
	base, ext, cur, err := extensionDocs(core, a)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rep := corecustom.Compare(base, ext, cur, pageGrouper)
	writeJSON(w, http.StatusOK, PageComparison{
		BaseVersion: int(a.BaseVersion.Int64), CoreVersion: core.Version,
		UpgradeAvailable: int(a.BaseVersion.Int64) < core.Version,
		Customizations:   nonNilGroups(rep.Customizations), CoreUpdates: nonNilGroups(rep.CoreUpdates),
	})
}

func nonNilGroups(g []corecustom.Group) []corecustom.Group {
	if g == nil {
		return []corecustom.Group{}
	}
	return g
}

// upgradeExtension rebases an extension onto the current core, carrying
// every customization except the groups in remove.
func upgradeExtension(core *PageStudioPage, a *pageAdoption, remove map[string]bool) (pageContent, bool, error) {
	base, ext, cur, err := extensionDocs(core, a)
	if err != nil {
		return pageContent{}, false, err
	}
	merged := corecustom.Merge(base, ext, cur, remove, pageGrouper)
	c, err := contentFromDoc(merged)
	if err != nil {
		return pageContent{}, false, err
	}
	return c, len(corecustom.Diff(cur, merged)) == 0, nil
}

// upgrade moves an extension to the current core version, keeping the
// customizations the tenant keeps and dropping those listed in remove.
// With no core change pending it just removes customizations. An
// extension left with no customizations goes back to vanilla.
func (h *PageStudioHandler) upgrade(w http.ResponseWriter, r *http.Request) {
	tenantID, core, a, ok := h.corePageRequest(w, r)
	if !ok {
		return
	}
	if a == nil || a.Mode != "extended" {
		http.Error(w, "this core page has no customizations to upgrade", http.StatusConflict)
		return
	}
	var req struct {
		Remove []string `json:"remove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	remove := make(map[string]bool, len(req.Remove))
	for _, id := range req.Remove {
		remove[id] = true
	}
	merged, vanilla, err := upgradeExtension(core, a, remove)
	if err != nil {
		http.Error(w, "failed to merge: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if vanilla {
		_, err = h.db.ExecContext(r.Context(), `
			UPDATE core_object_adoption SET mode = 'vanilla', base_version = NULL, base_snapshot = NULL, extension = NULL, updated_at = NOW(), updated_by = $4
			WHERE tenant_id = $1 AND object_type = $2 AND core_object_id = $3`, tenantID, corePageObjectType, core.ID, userOf(r))
		if err == nil {
			_, err = h.db.ExecContext(r.Context(), `
				DELETE FROM core_object_adoption WHERE tenant_id = $1 AND object_type = $2 AND core_object_id = $3 AND active`,
				tenantID, corePageObjectType, core.ID)
		}
	} else {
		var ext, base []byte
		if ext, err = json.Marshal(merged); err == nil {
			if base, err = json.Marshal(contentOf(core)); err == nil {
				_, err = h.db.ExecContext(r.Context(), `
					UPDATE core_object_adoption SET base_version = $4, base_snapshot = $5, extension = $6, updated_at = NOW(), updated_by = $7
					WHERE tenant_id = $1 AND object_type = $2 AND core_object_id = $3`,
					tenantID, corePageObjectType, core.ID, core.Version, base, ext, userOf(r))
			}
		}
	}
	if err != nil {
		http.Error(w, "failed to save upgrade: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.respondCore(w, r, tenantID, core)
}

// respondCore re-reads the adoption and writes the core page as the tenant
// now sees it.
func (h *PageStudioHandler) respondCore(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, core *PageStudioPage) {
	a, err := h.adoption(r.Context(), tenantID, core.ID)
	if err != nil {
		http.Error(w, "failed to reload customization: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fresh := *core
	presentCore(&fresh, a, true)
	writeJSON(w, http.StatusOK, fresh)
}

// pageGrouper maps a change in a page to the element a person keeps or
// removes: a widget, a layout container, a tab, a data source, a
// presentation rule, a piece of the app model, or the page's details.
// A widget's placement (its id in some container's children) groups with
// the widget, so "added a widget" is one decision.
func pageGrouper(p corecustom.Path, docs ...any) (string, string, string) {
	if len(p) == 0 {
		return "page", "page", "Page"
	}
	top := p[0].Value
	switch top {
	case "name", "description":
		return "page:details", "page", "Page name and description"
	case "components":
		if len(p) > 1 {
			return componentGroup(p[1].Value, docs)
		}
		return "components", "component", "Widgets"
	case "layout", "filterBar":
		scope := map[string]string{"layout": "Page layout", "filterBar": "Filter bar"}[top]
		return layoutGroup(p[1:], top, scope, docs)
	case "tabs":
		if len(p) == 1 {
			return "tabs", "tab", "Tabs"
		}
		if p[1].Kind == corecustom.SegOrder {
			return "tabs:order", "tab", "Tab order"
		}
		tabID := p[1].Value
		if len(p) > 2 && p[2].Value == "layout" {
			return layoutGroup(p[3:], "tab/"+tabID, "Tab "+tabLabel(tabID, docs), docs)
		}
		return "tab:" + tabID, "tab", "Tab " + tabLabel(tabID, docs)
	case "dataSources":
		if len(p) > 1 && p[1].Kind == corecustom.SegElem {
			return "dataSource:" + p[1].Value, "dataSource", "Data source " + quoted(field(docs, corecustom.Path{p[0], p[1]}, "name"), p[1].Value)
		}
		return "dataSources", "dataSource", "Data sources"
	case "presentationEvents":
		if len(p) > 1 && p[1].Kind == corecustom.SegElem {
			return "rule:" + p[1].Value, "rule", "Presentation rule " + quoted(field(docs, corecustom.Path{p[0], p[1]}, "label"), p[1].Value)
		}
		return "presentationEvents", "rule", "Presentation rules"
	case "app":
		if len(p) > 2 && p[2].Kind == corecustom.SegElem {
			return "app:" + p[1].Value + ":" + p[2].Value, "app", "App " + singular(p[1].Value) + " " + quoted("", p[2].Value)
		}
		if len(p) > 1 {
			return "app:" + p[1].Value, "app", "App " + p[1].Value
		}
		return "app", "app", "App model"
	}
	return p.String(), "other", p.String()
}

// layoutGroup groups a change inside one layout tree ({root, nodes}).
func layoutGroup(rest corecustom.Path, scopeID, scopeLabel string, docs []any) (string, string, string) {
	if len(rest) >= 2 && rest[0].Value == "nodes" {
		// nodes.<n>.children{member}: placing or removing a widget/container.
		if len(rest) >= 4 && rest[2].Value == "children" && rest[3].Kind == corecustom.SegMember {
			member := rest[3].Value
			if isComponent(member, docs) {
				return componentGroup(member, docs)
			}
			return "node:" + scopeID + "/" + member, "layout", scopeLabel + ": container " + member
		}
		return "node:" + scopeID + "/" + rest[1].Value, "layout", scopeLabel + ": container " + rest[1].Value
	}
	return "layout:" + scopeID, "layout", scopeLabel
}

func componentGroup(id string, docs []any) (string, string, string) {
	base := corecustom.Path{{Kind: corecustom.SegKey, Value: "components"}, {Kind: corecustom.SegKey, Value: id}}
	title := field(docs, append(base, corecustom.Seg{Kind: corecustom.SegKey, Value: "props"}), "title")
	if title == "" {
		title = field(docs, base, "label")
	}
	typ := field(docs, base, "type")
	label := "Widget " + quoted(title, id)
	if typ != "" {
		label += " (" + typ + ")"
	}
	return "component:" + id, "component", label
}

func isComponent(id string, docs []any) bool {
	for _, d := range docs {
		if _, ok := corecustom.Get(d, corecustom.Path{{Kind: corecustom.SegKey, Value: "components"}, {Kind: corecustom.SegKey, Value: id}}); ok {
			return true
		}
	}
	return false
}

func tabLabel(id string, docs []any) string {
	return quoted(field(docs, corecustom.Path{{Kind: corecustom.SegKey, Value: "tabs"}, {Kind: corecustom.SegElem, Value: id}}, "label"), id)
}

// field reads a string field of the object at path from the first
// document that has one.
func field(docs []any, path corecustom.Path, name string) string {
	for _, d := range docs {
		if v, ok := corecustom.Get(d, path); ok {
			if m, ok := v.(map[string]any); ok {
				if s, ok := m[name].(string); ok && s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func quoted(label, fallback string) string {
	if label == "" {
		label = fallback
	}
	return "“" + label + "”"
}

func singular(s string) string {
	return strings.TrimSuffix(s, "s")
}
