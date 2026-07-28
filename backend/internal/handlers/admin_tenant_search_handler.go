package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
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

func (h *AdminTenantSearchHandler) GetTenantScope(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "tenant search handler not configured", http.StatusInternalServerError)
		return
	}

	tenantIDStr := chi.URLParam(r, "tenantID")
	if _, err := uuid.Parse(tenantIDStr); err != nil {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	if err := db.RequireVerifiedTenantFromCtx(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	var result struct {
		TenantID  string             `json:"tenant_id"`
		Instances []TenantScopeNode  `json:"instances"`
	}

	err := db.WithTenantTransaction(r.Context(), h.db, tenantIDStr, func(tx *sql.Tx) error {
		instances, err := h.getTenantScopeTx(r.Context(), tx, tenantIDStr)
		if err != nil {
			return err
		}
		result.TenantID = tenantIDStr
		result.Instances = instances
		return nil
	})
	if err != nil {
		http.Error(w, "failed to fetch tenant scope: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *AdminTenantSearchHandler) getTenantScopeTx(ctx context.Context, tx *sql.Tx, tenantID string) ([]TenantScopeNode, error) {
	dbQ := func(query string, args ...interface{}) (*sql.Rows, error) {
		return tx.QueryContext(ctx, query, args...)
	}

	rows, err := dbQ(`SELECT id, COALESCE(display_name, name) FROM tenant_instance WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query instances: %w", err)
	}
	defer rows.Close()

	var instances []TenantScopeNode
	var instanceIDs []string
	for rows.Next() {
		var inst TenantScopeNode
		if err := rows.Scan(&inst.ID, &inst.Name); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		inst.Type = "instance"
		instances = append(instances, inst)
		instanceIDs = append(instanceIDs, inst.ID)
	}

	productsByInstance := map[string][]TenantScopeNode{}
	var productIDs []string
	if len(instanceIDs) > 0 {
		prows, err := dbQ(`SELECT id, COALESCE(display_name, ''), tenant_instance_id FROM tenant_product WHERE tenant_instance_id = ANY($1)`,
			pq.Array(instanceIDs))
		if err != nil {
			return nil, fmt.Errorf("failed to query products: %w", err)
		}
		defer prows.Close()
		for prows.Next() {
			var p TenantScopeNode
			var instID string
			if err := prows.Scan(&p.ID, &p.Name, &instID); err != nil {
				return nil, fmt.Errorf("scan failed: %w", err)
			}
			p.Type = "product"
			productsByInstance[instID] = append(productsByInstance[instID], p)
			productIDs = append(productIDs, p.ID)
		}
	}

	datasourcesByProduct := map[string][]TenantScopeNode{}
	if len(productIDs) > 0 {
		dsrows, err := dbQ(`SELECT id, COALESCE(source_name, ''), tenant_product_id FROM tenant_product_datasource WHERE tenant_product_id = ANY($1)`,
			pq.Array(productIDs))
		if err != nil {
			return nil, fmt.Errorf("failed to query datasources: %w", err)
		}
		defer dsrows.Close()
		for dsrows.Next() {
			var d TenantScopeNode
			var prodID string
			if err := dsrows.Scan(&d.ID, &d.Name, &prodID); err != nil {
				return nil, fmt.Errorf("scan failed: %w", err)
			}
			d.Type = "datasource"
			datasourcesByProduct[prodID] = append(datasourcesByProduct[prodID], d)
		}
	}

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

	return instances, nil
}