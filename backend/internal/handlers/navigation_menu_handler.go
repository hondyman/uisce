package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/jmoiron/sqlx"
)

// NavigationMenuNode is the wire shape for one node of the tenant's
// navigation tree (navigation_menu_nodes). Children is populated only by
// the tree-shaped list endpoint; flat reads/writes omit it.
type NavigationMenuNode struct {
	ID                  uuid.UUID             `json:"id" db:"id"`
	TenantID            uuid.UUID             `json:"-" db:"tenant_id"`
	ParentID            *uuid.UUID            `json:"parentId,omitempty" db:"parent_id"`
	NodeKey             string                `json:"nodeKey" db:"node_key"`
	Label               string                `json:"label" db:"label"`
	Icon                *string               `json:"icon,omitempty" db:"icon"`
	TargetPageKey       *string               `json:"targetPageKey,omitempty" db:"target_page_key"`
	DisplayOrder        int                   `json:"displayOrder" db:"display_order"`
	RequiredEntitlement string                `json:"requiredEntitlement" db:"required_entitlement"`
	Children            []*NavigationMenuNode `json:"children,omitempty" db:"-"`
	// Inherited: a gold-copy node seen from another tenant - shown, never
	// editable there. A tenant may add its own nodes under it.
	Inherited bool `json:"inherited,omitempty" db:"-"`
	// Hidden: this tenant has switched a gold-copy entry (or its folder) off.
	// Only the Menu Designer sees hidden entries (listTree?includeHidden=true).
	Hidden bool `json:"hidden,omitempty" db:"hidden"`
}

// loadMenuNodes returns the tenant's own menu nodes plus the gold copy's,
// which every tenant inherits read-only (like core pages). Entries that
// open a core page the tenant has switched off are left out, and so are
// gold-copy entries the tenant has hidden (with everything under them).
func loadMenuNodes(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID) ([]NavigationMenuNode, error) {
	return queryMenuNodes(ctx, db, tenantID, false)
}

// queryMenuNodes is loadMenuNodes, optionally keeping the hidden entries
// (flagged Hidden) so the Menu Designer can show them and bring them back.
func queryMenuNodes(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, includeHidden bool) ([]NavigationMenuNode, error) {
	gold := goldcopy.ResolveTenantID(ctx, db)
	var flat []NavigationMenuNode
	err := db.SelectContext(ctx, &flat, `
		SELECT n.id, n.tenant_id, n.parent_id, n.node_key, n.label, n.icon, n.target_page_key, n.display_order, n.required_entitlement,
		       (o.node_id IS NOT NULL) AS hidden
		FROM navigation_menu_nodes n
		LEFT JOIN navigation_menu_placement_overrides o ON o.node_id = n.id AND o.tenant_id = $1
		WHERE (n.tenant_id = $1 OR n.tenant_id = $2)
		  AND (n.target_page_key IS NULL OR n.target_page_key NOT IN (
		        SELECT p.slug FROM core_object_adoption a
		        JOIN page_definitions p ON p.id = a.core_object_id
		        WHERE a.tenant_id = $1 AND a.object_type = 'page' AND NOT a.active))
		ORDER BY n.display_order, n.label
	`, tenantID, gold)
	if err != nil {
		return nil, err
	}
	for i := range flat {
		flat[i].Inherited = flat[i].TenantID != tenantID
	}
	return applyHidden(flat, includeHidden), nil
}

// applyHidden extends a hidden entry to everything under it: a hidden folder
// takes its entries with it. Without this, buildMenuTree would promote an
// orphaned child to a root. Hidden entries are dropped unless includeHidden.
func applyHidden(flat []NavigationMenuNode, includeHidden bool) []NavigationMenuNode {
	children := make(map[uuid.UUID][]int, len(flat))
	for i := range flat {
		if flat[i].ParentID != nil {
			children[*flat[i].ParentID] = append(children[*flat[i].ParentID], i)
		}
	}
	var mark func(i int)
	mark = func(i int) {
		for _, c := range children[flat[i].ID] {
			if !flat[c].Hidden {
				flat[c].Hidden = true
				mark(c)
			}
		}
	}
	for i := range flat {
		if flat[i].Hidden {
			mark(i)
		}
	}
	if includeHidden {
		return flat
	}
	out := flat[:0]
	for _, n := range flat {
		if !n.Hidden {
			out = append(out, n)
		}
	}
	return out
}

// buildMenuTree nests flat nodes under their parents; nodes whose parent is
// not in the set become roots.
func buildMenuTree(flat []NavigationMenuNode) []*NavigationMenuNode {
	byID := make(map[uuid.UUID]*NavigationMenuNode, len(flat))
	for i := range flat {
		flat[i].Children = []*NavigationMenuNode{}
		byID[flat[i].ID] = &flat[i]
	}
	roots := make([]*NavigationMenuNode, 0)
	for i := range flat {
		n := &flat[i]
		if n.ParentID != nil {
			if parent, ok := byID[*n.ParentID]; ok {
				parent.Children = append(parent.Children, n)
				continue
			}
		}
		roots = append(roots, n)
	}
	sort.SliceStable(roots, func(i, j int) bool { return roots[i].DisplayOrder < roots[j].DisplayOrder })
	return roots
}

// PageMenuPlacement is where a page sits on the menu: the labels from the
// top of the menu down to its entry.
type PageMenuPlacement struct {
	NodeID    uuid.UUID `json:"nodeId"`
	Path      []string  `json:"path"`
	Inherited bool      `json:"inherited,omitempty"`
}

// menuPlacements maps a page slug to every menu entry that opens it.
func menuPlacements(flat []NavigationMenuNode) map[string][]PageMenuPlacement {
	byID := make(map[uuid.UUID]*NavigationMenuNode, len(flat))
	for i := range flat {
		byID[flat[i].ID] = &flat[i]
	}
	out := map[string][]PageMenuPlacement{}
	for i := range flat {
		n := &flat[i]
		if n.TargetPageKey == nil || *n.TargetPageKey == "" {
			continue
		}
		path := []string{n.Label}
		seen := map[uuid.UUID]bool{n.ID: true}
		for p := n.ParentID; p != nil && !seen[*p]; {
			parent, ok := byID[*p]
			if !ok {
				break
			}
			seen[*p] = true
			path = append([]string{parent.Label}, path...)
			p = parent.ParentID
		}
		out[*n.TargetPageKey] = append(out[*n.TargetPageKey], PageMenuPlacement{NodeID: n.ID, Path: path, Inherited: n.Inherited})
	}
	return out
}

// parentAllowed: a node may hang under one of the tenant's own nodes or an
// inherited gold-copy node - never another tenant's.
func (h *NavigationMenuHandler) parentAllowed(ctx context.Context, tenantID uuid.UUID, parent *uuid.UUID) (bool, error) {
	if parent == nil {
		return true, nil
	}
	var owner uuid.UUID
	err := h.db.GetContext(ctx, &owner, `SELECT tenant_id FROM navigation_menu_nodes WHERE id = $1`, *parent)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if owner == tenantID {
		return true, nil
	}
	gold := goldcopy.ResolveTenantID(ctx, h.db)
	return gold != uuid.Nil && owner == gold, nil
}

type NavigationMenuHandler struct {
	db *sqlx.DB
}

func NewNavigationMenuHandler(db *sqlx.DB) *NavigationMenuHandler {
	return &NavigationMenuHandler{db: db}
}

func (h *NavigationMenuHandler) RegisterRoutes(r chi.Router) {
	r.Route("/navigation-menu", func(r chi.Router) {
		r.Get("/", h.listTree)
		r.Post("/", h.create)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Put("/{id}/hidden", h.setHidden)
	})
}

// listTree returns every node for the tenant assembled into a tree
// (top-level nodes with nested children), the shape the Menu Designer
// and the consumer-facing nav both want - a flat list would make both
// re-derive the same parent/child grouping client-side.
func (h *NavigationMenuHandler) listTree(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	includeHidden := r.URL.Query().Get("includeHidden") == "true"
	flat, err := queryMenuNodes(r.Context(), h.db, tenantID, includeHidden)
	if err != nil {
		http.Error(w, "failed to list navigation menu: "+err.Error(), http.StatusInternalServerError)
		return
	}
	roots := buildMenuTree(flat)
	writeJSON(w, http.StatusOK, roots)
}

type navMenuUpsertRequest struct {
	ParentID            *uuid.UUID `json:"parentId"`
	NodeKey             string     `json:"nodeKey"`
	Label               string     `json:"label"`
	Icon                *string    `json:"icon"`
	TargetPageKey       *string    `json:"targetPageKey"`
	DisplayOrder        int        `json:"displayOrder"`
	RequiredEntitlement string     `json:"requiredEntitlement"`
}

func (h *NavigationMenuHandler) create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req navMenuUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Label == "" || req.NodeKey == "" {
		http.Error(w, "nodeKey and label are required", http.StatusBadRequest)
		return
	}
	if req.RequiredEntitlement == "" {
		req.RequiredEntitlement = "BASE_USER"
	}
	if ok, err := h.parentAllowed(r.Context(), tenantID, req.ParentID); err != nil || !ok {
		http.Error(w, "parent menu node not found", http.StatusBadRequest)
		return
	}

	id := uuid.New()
	var node NavigationMenuNode
	err := h.db.GetContext(r.Context(), &node, `
		INSERT INTO navigation_menu_nodes (id, tenant_id, parent_id, node_key, label, icon, target_page_key, display_order, required_entitlement)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, parent_id, node_key, label, icon, target_page_key, display_order, required_entitlement
	`, id, tenantID, req.ParentID, req.NodeKey, req.Label, req.Icon, req.TargetPageKey, req.DisplayOrder, req.RequiredEntitlement)
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "a menu node with this key already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create menu node: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, node)
}

func (h *NavigationMenuHandler) update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req navMenuUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Label == "" || req.NodeKey == "" {
		http.Error(w, "nodeKey and label are required", http.StatusBadRequest)
		return
	}
	if req.RequiredEntitlement == "" {
		req.RequiredEntitlement = "BASE_USER"
	}
	// A node may not become its own ancestor - the FK would happily
	// accept a self-parent and orphan the tree into a cycle that
	// listTree's single-pass grouping can't detect or terminate.
	if req.ParentID != nil && *req.ParentID == id {
		http.Error(w, "a menu node cannot be its own parent", http.StatusBadRequest)
		return
	}
	if ok, err := h.parentAllowed(r.Context(), tenantID, req.ParentID); err != nil || !ok {
		http.Error(w, "parent menu node not found", http.StatusBadRequest)
		return
	}
	// Moving a node under one of its own descendants would close a loop the
	// same way (the self-parent check above only catches the direct case).
	if req.ParentID != nil {
		within, err := h.isWithin(r.Context(), *req.ParentID, id)
		if err != nil {
			http.Error(w, "failed to check menu position: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if within {
			http.Error(w, "a menu node cannot be moved under its own descendant", http.StatusBadRequest)
			return
		}
	}

	var node NavigationMenuNode
	err = h.db.GetContext(r.Context(), &node, `
		UPDATE navigation_menu_nodes
		SET parent_id = $1, node_key = $2, label = $3, icon = $4, target_page_key = $5,
		    display_order = $6, required_entitlement = $7
		WHERE id = $8 AND tenant_id = $9
		RETURNING id, tenant_id, parent_id, node_key, label, icon, target_page_key, display_order, required_entitlement
	`, req.ParentID, req.NodeKey, req.Label, req.Icon, req.TargetPageKey, req.DisplayOrder, req.RequiredEntitlement, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "menu node not found", http.StatusNotFound)
			return
		}
		if isUniqueViolation(err) {
			http.Error(w, "a menu node with this key already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to update menu node: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

// isWithin reports whether candidate is node or sits anywhere under it, by
// walking up candidate's parent_id chain. UNION (not UNION ALL) keeps the
// walk finite even if the stored tree already contains a loop.
func (h *NavigationMenuHandler) isWithin(ctx context.Context, candidate, node uuid.UUID) (bool, error) {
	var within bool
	err := h.db.GetContext(ctx, &within, `
		WITH RECURSIVE up AS (
			SELECT id, parent_id FROM navigation_menu_nodes WHERE id = $1
			UNION
			SELECT n.id, n.parent_id FROM navigation_menu_nodes n JOIN up ON n.id = up.parent_id
		)
		SELECT EXISTS (SELECT 1 FROM up WHERE id = $2)
	`, candidate, node)
	return within, err
}

type navMenuHiddenRequest struct {
	Hidden bool `json:"hidden"`
}

// setHidden switches a gold-copy entry off (or back on) for this tenant. The
// override is a row in the tenant's own table; the gold-copy row is never
// written, so the gold copy stays the only place core menu structure changes.
func (h *NavigationMenuHandler) setHidden(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req navMenuHiddenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var owner uuid.UUID
	err = h.db.GetContext(r.Context(), &owner, `SELECT tenant_id FROM navigation_menu_nodes WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		http.Error(w, "menu node not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to read menu node: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if owner == tenantID {
		http.Error(w, "only core entries can be hidden; delete your own entry instead", http.StatusBadRequest)
		return
	}
	if gold := goldcopy.ResolveTenantID(r.Context(), h.db); gold == uuid.Nil || owner != gold {
		http.Error(w, "menu node not found", http.StatusNotFound)
		return
	}
	if req.Hidden {
		_, err = h.db.ExecContext(r.Context(), `
			INSERT INTO navigation_menu_placement_overrides (tenant_id, node_id)
			VALUES ($1, $2)
			ON CONFLICT (tenant_id, node_id) DO NOTHING
		`, tenantID, id)
	} else {
		_, err = h.db.ExecContext(r.Context(), `
			DELETE FROM navigation_menu_placement_overrides WHERE tenant_id = $1 AND node_id = $2
		`, tenantID, id)
	}
	if err != nil {
		http.Error(w, "failed to update menu node visibility: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *NavigationMenuHandler) delete(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	// ON DELETE CASCADE on parent_id means this also removes descendants -
	// matches the Menu Designer's "delete a folder, its pages go with it"
	// expectation, not a partial/dangling tree.
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM navigation_menu_nodes WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		http.Error(w, "failed to delete menu node: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "menu node not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
