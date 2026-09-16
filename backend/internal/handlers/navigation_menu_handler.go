package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// NavigationMenuNode is the wire shape for one node of the tenant's
// navigation tree (navigation_menu_nodes). Children is populated only by
// the tree-shaped list endpoint; flat reads/writes omit it.
type NavigationMenuNode struct {
	ID                  uuid.UUID              `json:"id" db:"id"`
	TenantID            uuid.UUID              `json:"-" db:"tenant_id"`
	ParentID            *uuid.UUID             `json:"parentId,omitempty" db:"parent_id"`
	NodeKey             string                 `json:"nodeKey" db:"node_key"`
	Label               string                 `json:"label" db:"label"`
	Icon                *string                `json:"icon,omitempty" db:"icon"`
	TargetPageKey       *string                `json:"targetPageKey,omitempty" db:"target_page_key"`
	DisplayOrder        int                    `json:"displayOrder" db:"display_order"`
	RequiredEntitlement string                 `json:"requiredEntitlement" db:"required_entitlement"`
	Children            []*NavigationMenuNode  `json:"children,omitempty" db:"-"`
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
	var flat []NavigationMenuNode
	err := h.db.SelectContext(r.Context(), &flat, `
		SELECT id, tenant_id, parent_id, node_key, label, icon, target_page_key, display_order, required_entitlement
		FROM navigation_menu_nodes
		WHERE tenant_id = $1
		ORDER BY display_order, label
	`, tenantID)
	if err != nil {
		http.Error(w, "failed to list navigation menu: "+err.Error(), http.StatusInternalServerError)
		return
	}

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

	writeJSON(w, http.StatusOK, roots)
}

type navMenuUpsertRequest struct {
	ParentID             *uuid.UUID `json:"parentId"`
	NodeKey              string     `json:"nodeKey"`
	Label                string     `json:"label"`
	Icon                 *string    `json:"icon"`
	TargetPageKey        *string    `json:"targetPageKey"`
	DisplayOrder         int        `json:"displayOrder"`
	RequiredEntitlement  string     `json:"requiredEntitlement"`
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
