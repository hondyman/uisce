package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AdminTenantSearchHandler provides lightweight, impersonation-picker-facing
// tenant lookup endpoints. Kept separate from AdminTenantHandler so the picker
// doesn't drag in the full admin tenant CRUD surface area.
type AdminTenantSearchHandler struct {
	db *sql.DB
}

// NewAdminTenantSearchHandler constructs a search handler bound to a DB pool.
func NewAdminTenantSearchHandler(db *sql.DB) *AdminTenantSearchHandler {
	return &AdminTenantSearchHandler{db: db}
}

// TenantSearchResult is a lightweight row for the impersonation picker.
type TenantSearchResult struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Code          string `json:"code,omitempty"`
	Region        string `json:"region,omitempty"`
	Plan          string `json:"plan,omitempty"`
	IsSuspended   bool   `json:"is_suspended"`
	InstanceCount int    `json:"instance_count"`
}

// SearchTenants handles GET /api/admin/tenants/search?q=<text>&limit=20
// Powers the impersonation picker's left-pane tenant list.
//
// Query params:
//   - q (optional, min 2 chars when provided): case-insensitive prefix/contains match on name/code
//   - limit (optional, default 20, max 50): max results to return
//   - suspended (optional, "true"/"false"): filter by suspended state
func (h *AdminTenantSearchHandler) SearchTenants(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "tenant search handler not configured", http.StatusInternalServerError)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q != "" && utf8.RuneCountInString(q) < 2 {
		http.Error(w, "q must be at least 2 characters", http.StatusBadRequest)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	includeSuspended := true
	if r.URL.Query().Get("suspended") == "false" {
		includeSuspended = false
	}

	args := []interface{}{}
	where := []string{}
	if q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, "(LOWER(t.name) LIKE $"+strconv.Itoa(len(args))+" OR LOWER(COALESCE(t.code, '')) LIKE $"+strconv.Itoa(len(args))+")")
	}
	if !includeSuspended {
		where = append(where, "(t.is_suspended IS NULL OR t.is_suspended = false)")
	}

	query := `SELECT t.id, t.name, t.code, t.region, t.plan,
	                 COALESCE(t.is_suspended, false) as is_suspended,
	                 (SELECT COUNT(*) FROM tenant_instance ti WHERE ti.tenant_id = t.id) AS instance_count
	          FROM tenants t`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY t.name ASC LIMIT " + strconv.Itoa(limit)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		http.Error(w, "search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results := make([]TenantSearchResult, 0, limit)
	for rows.Next() {
		var r TenantSearchResult
		var code, region, plan sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &code, &region, &plan, &r.IsSuspended, &r.InstanceCount); err != nil {
			http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		r.Code = code.String
		r.Region = region.String
		r.Plan = plan.String
		results = append(results, r)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

// TenantScopeNode represents one node in the tenant's instance/product/datasource tree.
type TenantScopeNode struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Type     string           `json:"type"` // "instance" | "product" | "datasource"
	Children []TenantScopeNode `json:"children,omitempty"`
}

// GetTenantScope handles GET /api/admin/tenants/{tenantID}/scope
// Returns the hierarchical instance → product → datasource tree for a tenant.
// Used by the impersonation picker to render the right-pane scope selector.
func (h *AdminTenantSearchHandler) GetTenantScope(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "tenant search handler not configured", http.StatusInternalServerError)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Fetch all instances for the tenant in one query.
	instanceRows, err := h.db.QueryContext(r.Context(),
		`SELECT id, COALESCE(display_name, name) FROM tenant_instance WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		http.Error(w, "failed to query instances: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer instanceRows.Close()

	instances := []TenantScopeNode{}
	instanceIDs := []string{}
	for instanceRows.Next() {
		var inst TenantScopeNode
		if err := instanceRows.Scan(&inst.ID, &inst.Name); err != nil {
			http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		inst.Type = "instance"
		instances = append(instances, inst)
		instanceIDs = append(instanceIDs, inst.ID)
	}

	// Fetch all products keyed by instance.
	productsByInstance := map[string][]TenantScopeNode{}
	productIDs := []string{}
	if len(instanceIDs) > 0 {
		productRows, err := h.db.QueryContext(r.Context(),
			`SELECT id, COALESCE(display_name, ''), tenant_instance_id FROM tenant_product WHERE tenant_instance_id = ANY($1)`,
			pq.Array(instanceIDs))
		if err != nil {
			http.Error(w, "failed to query products: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer productRows.Close()
		for productRows.Next() {
			var p TenantScopeNode
			var instID string
			if err := productRows.Scan(&p.ID, &p.Name, &instID); err != nil {
				http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
			p.Type = "product"
			productsByInstance[instID] = append(productsByInstance[instID], p)
			productIDs = append(productIDs, p.ID)
		}
	}

	// Fetch all datasources keyed by product.
	datasourcesByProduct := map[string][]TenantScopeNode{}
	if len(productIDs) > 0 {
		dsRows, err := h.db.QueryContext(r.Context(),
			`SELECT id, COALESCE(source_name, ''), tenant_product_id FROM tenant_product_datasource WHERE tenant_product_id = ANY($1)`,
			pq.Array(productIDs))
		if err != nil {
			http.Error(w, "failed to query datasources: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer dsRows.Close()
		for dsRows.Next() {
			var d TenantScopeNode
			var prodID string
			if err := dsRows.Scan(&d.ID, &d.Name, &prodID); err != nil {
				http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
			d.Type = "datasource"
			datasourcesByProduct[prodID] = append(datasourcesByProduct[prodID], d)
		}
	}

	// Assemble the tree.
	for i := range instances {
		if products, ok := productsByInstance[instances[i].ID]; ok {
			for j := range products {
				if ds, ok := datasourcesByProduct[products[j].ID]; ok {
					products[j].Children = ds
				}
			}
			instances[i].Children = products
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID.String(),
		"instances": instances,
	})
}