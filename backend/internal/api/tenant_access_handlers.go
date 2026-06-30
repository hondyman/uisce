package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// TenantAccessHandlers provides endpoints for tenant access control
type TenantAccessHandlers struct {
	DB *sql.DB
}

// NewTenantAccessHandlers creates a new TenantAccessHandlers instance
func NewTenantAccessHandlers(db *sql.DB) *TenantAccessHandlers {
	return &TenantAccessHandlers{DB: db}
}

// RegisterRoutes registers the tenant access routes
func (h *TenantAccessHandlers) RegisterRoutes(r chi.Router) {
	r.Get("/tenants/accessible", h.listAccessibleTenants)
	r.Get("/tenants/debug", h.listAccessibleTenants)
	r.Get("/tenants/all", h.listAllTenants)

	// Connection sync handler moved to handlers.ConnectionSyncHandler
	// r.Post("/tenants/{tenantId}/sync-connections", syncHandler.SyncConnectionsFromGoldCopy)
}

// TenantResponse represents a tenant in the API response
type TenantResponse struct {
	ID             string             `json:"id"`
	DisplayName    string             `json:"display_name"`
	Name           string             `json:"name,omitempty"`
	Description    *string            `json:"description,omitempty"`
	IsActive       bool               `json:"is_active"`
	GoldCopy       bool               `json:"gold_copy"`
	Region         string             `json:"region"`
	AllowedRegions []string           `json:"allowed_regions"`
	Instances      []InstanceResponse `json:"tenant_instances"`
}

// InstanceResponse represents a tenant instance in the API response
type InstanceResponse struct {
	ID          string            `json:"id"`
	DisplayName string            `json:"display_name"`
	Name        string            `json:"instance_name,omitempty"`
	Description *string           `json:"description,omitempty"`
	IsActive    bool              `json:"is_active"`
	URL         *string           `json:"url,omitempty"`
	TenantID    string            `json:"tenant_id"`
	Products    []ProductResponse `json:"tenant_products"`
}

// ProductResponse represents a product in the API response
type ProductResponse struct {
	ID               string               `json:"id"`
	Version          float64              `json:"version"`
	TenantInstanceID string               `json:"datasource_id"`
	AlphaProductID   string               `json:"alpha_product_id"`
	AlphaProduct     *AlphaProductInfo    `json:"alpha_product,omitempty"`
	Datasources      []DatasourceResponse `json:"tenant_product_datasources"`
}

// AlphaProductInfo contains core product information
type AlphaProductInfo struct {
	ID          string  `json:"id"`
	ProductName string  `json:"product_name"`
	ProductCode *string `json:"product_code"` // Nullable since it's NULL::text in query
	IsActive    bool    `json:"is_active"`
}

// DatasourceResponse represents a datasource in the API response
type DatasourceResponse struct {
	ID                string               `json:"id"`
	AlphaDatasourceID string               `json:"alpha_datasource_id"`
	IsActive          bool                 `json:"is_active"`
	SourceName        string               `json:"source_name"`
	AlphaDatasource   *AlphaDatasourceInfo `json:"alpha_datasource,omitempty"`
}

// AlphaDatasourceInfo contains core datasource information
type AlphaDatasourceInfo struct {
	ID             string `json:"id"`
	DatasourceName string `json:"datasource_name"`
	DatasourceType string `json:"datasource_type"`
	DatasourceCode string `json:"datasource_code,omitempty"`
}

// listAccessibleTenants returns tenants the current user can access
// Platform operators see all tenants
// Tenant admins/users see only their assigned tenants
func (h *TenantAccessHandlers) listAccessibleTenants(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("PANIC in listAccessibleTenants: %v\n", r)
			// Print stack trace
			// debug.PrintStack() // debug package check
			http.Error(w, "Panic", http.StatusInternalServerError)
		}
	}()

	// Get user info from headers (set by auth middleware)
	userRole := r.Header.Get("X-User-Role")
	userID := r.Header.Get("X-User-ID")
	isCoreAdmin := r.Header.Get("X-Is-Core-Admin") == "true"

	fmt.Printf("[DEBUG] listAccessibleTenants: UserID=%s Role=%s CoreAdmin=%v\n", userID, userRole, isCoreAdmin)

	// Platform operators (core admins and Uisce internal support tiers) see all tenants.
	isPlatformOperator := isCoreAdmin ||
		userRole == "platform_operator" ||
		userRole == "admin" ||
		userRole == "global_admin" ||
		userRole == "helpdesk" ||
		userRole == "professional_services" ||
		strings.Contains(r.Header.Get("X-User-Permissions"), "platform:operator")

	if isPlatformOperator {
		h.listAllTenants(w, r)
		return
	}

	// For non-operators, filter by tenant assignments
	// This requires a user_tenant_assignments table
	tenants, err := h.getTenantsByUser(r, userID)
	if err != nil {
		fmt.Printf("[DEBUG] getTenantsByUser error: %v\n", err)
		http.Error(w, "Failed to fetch accessible tenants: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[DEBUG] listAccessibleTenants found %d tenants for user %s\n", len(tenants), userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

// listAllTenants returns all tenants with full hierarchy
func (h *TenantAccessHandlers) listAllTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.getAllTenantsInternal(r.Context(), nil) // nil means fetch all
	if err != nil {
		http.Error(w, "Failed to query tenants: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

func (h *TenantAccessHandlers) getAllTenantsInternal(ctx context.Context, targetTenantID *string) ([]TenantResponse, error) {
	// 1. Query tenants (optionally filtered)
	var args []interface{}
	query := `
		SELECT id, COALESCE(display_name, name, '') as display_name,
		       COALESCE(name, '') as name, description,
		       COALESCE(is_active, true) as is_active,
		       COALESCE(gold_copy, false) as gold_copy,
		       COALESCE(region, 'us-west') as region,
		       COALESCE(allowed_regions, '[]'::jsonb) as allowed_regions
		FROM tenants WHERE 1=1
	`
	if targetTenantID != nil {
		query += " AND id = $1"
		args = append(args, *targetTenantID)
	}
	query += " ORDER BY display_name"

	tenantRows, err := h.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer tenantRows.Close()

	var tenants []TenantResponse
	for tenantRows.Next() {
		var t TenantResponse
		var allowedRegionsJSON []byte
		if err := tenantRows.Scan(&t.ID, &t.DisplayName, &t.Name, &t.Description, &t.IsActive, &t.GoldCopy, &t.Region, &allowedRegionsJSON); err != nil {
			return nil, err
		}
		if len(allowedRegionsJSON) > 0 {
			if err := json.Unmarshal(allowedRegionsJSON, &t.AllowedRegions); err != nil {
				fmt.Printf("Error unmarshaling allowed_regions for tenant %s: %v\n", t.ID, err)
				t.AllowedRegions = []string{}
			}
		} else {
			t.AllowedRegions = []string{}
		}
		t.Instances = []InstanceResponse{}
		tenants = append(tenants, t)
	}

	if len(tenants) == 0 {
		return []TenantResponse{}, nil
	}

	// 2. Query instances
	instanceQuery := `
		SELECT id, COALESCE(display_name, instance_name, '') as display_name,
		       COALESCE(instance_name, '') as instance_name, NULL::text as description,
		       COALESCE(is_active, true) as is_active, url, tenant_id
		FROM tenant_instance WHERE 1=1
	`
	var iArgs []interface{}
	if targetTenantID != nil {
		instanceQuery += " AND tenant_id = $1"
		iArgs = append(iArgs, *targetTenantID)
	}
	instanceQuery += " ORDER BY display_name"

	instanceRows, err := h.DB.QueryContext(ctx, instanceQuery, iArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query instances: %w", err)
	}
	defer instanceRows.Close()

	instanceMap := make(map[string][]InstanceResponse)
	instanceIDs := []string{}
	for instanceRows.Next() {
		var i InstanceResponse
		if err := instanceRows.Scan(&i.ID, &i.DisplayName, &i.Name, &i.Description, &i.IsActive, &i.URL, &i.TenantID); err != nil {
			return nil, err
		}
		i.Products = []ProductResponse{}
		instanceMap[i.TenantID] = append(instanceMap[i.TenantID], i)
		instanceIDs = append(instanceIDs, i.ID)
	}

	// 3. Query products by tenant (schema: tenant_product.tenant_id)
	productQuery := `
		SELECT tp.id, tp.version, tp.alpha_product_id,
		       ap.id as ap_id, COALESCE(ap.product_name, '') as product_name,
		       NULL::text as product_code, COALESCE(ap.is_active, true) as ap_is_active
		FROM tenant_product tp
		LEFT JOIN alpha_product ap ON ap.id = tp.alpha_product_id
		WHERE 1=1
	`
	var pArgs []interface{}
	if targetTenantID != nil {
		productQuery += " AND tp.tenant_id = $1"
		pArgs = append(pArgs, *targetTenantID)
	}
	productQuery += " ORDER BY ap.product_name"

	productRows, err := h.DB.QueryContext(ctx, productQuery, pArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer productRows.Close()

	productMap := make(map[string]ProductResponse)
	for productRows.Next() {
		var p ProductResponse
		var ap AlphaProductInfo
		if err := productRows.Scan(&p.ID, &p.Version, &p.AlphaProductID,
			&ap.ID, &ap.ProductName, &ap.ProductCode, &ap.IsActive); err != nil {
			return nil, err
		}
		p.AlphaProduct = &ap
		p.Datasources = []DatasourceResponse{}
		productMap[p.ID] = p
	}

	// 4. Query datasources linking products to instances
	dsQuery := `
		SELECT tpd.id, COALESCE(tpd.is_active, true) as is_active,
		       COALESCE(tpd.source_name, '') as source_name, tpd.tenant_product_id,
		       tpd.tenant_instance_id
		FROM tenant_product_datasource tpd
		WHERE 1=1
	`
	var dArgs []interface{}
	if targetTenantID != nil {
		dsQuery += " AND tpd.tenant_instance_id IN (SELECT id FROM tenant_instance WHERE tenant_id = $1)"
		dArgs = append(dArgs, *targetTenantID)
	}
	dsQuery += " ORDER BY tpd.source_name"

	dsRows, err := h.DB.QueryContext(ctx, dsQuery, dArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query datasources: %w", err)
	}
	defer dsRows.Close()

	// instanceID -> []ProductResponse
	instanceProducts := make(map[string][]ProductResponse)
	for dsRows.Next() {
		var ds DatasourceResponse
		var productID string
		var instanceID sql.NullString
		if err := dsRows.Scan(&ds.ID, &ds.IsActive, &ds.SourceName, &productID, &instanceID); err != nil {
			return nil, err
		}
		if !instanceID.Valid || instanceID.String == "" {
			continue
		}
		if product, ok := productMap[productID]; ok {
			product.Datasources = append(product.Datasources, ds)
			// Ensure each product appears once per instance.
			existing := false
			for _, ep := range instanceProducts[instanceID.String] {
				if ep.ID == product.ID {
					existing = true
					break
				}
			}
			if !existing {
				instanceProducts[instanceID.String] = append(instanceProducts[instanceID.String], product)
			}
		}
	}

	// Assemble the hierarchy
	for i := range tenants {
		if instances, ok := instanceMap[tenants[i].ID]; ok {
			for j := range instances {
				if products, ok := instanceProducts[instances[j].ID]; ok {
					instances[j].Products = products
				}
			}
			tenants[i].Instances = instances
		}
	}

	return tenants, nil
}

// getTenantsByUser returns tenants accessible to a specific user
func (h *TenantAccessHandlers) getTenantsByUser(r *http.Request, userID string) ([]TenantResponse, error) {
	ctx := r.Context()

	// Defensive: requests without a resolved user should not crash the DB query.
	if strings.TrimSpace(userID) == "" {
		fmt.Printf("[DEBUG] getTenantsByUser called with empty userID. Returning empty tenant list.\n")
		return []TenantResponse{}, nil
	}

	// 1. Check for explicit tenant binding in users table
	var tenantID sql.NullString
	err := h.DB.QueryRowContext(ctx, "SELECT tenant_id FROM users WHERE id = $1", userID).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			// User has not been provisioned in the platform yet; treat as no accessible tenants.
			fmt.Printf("[DEBUG] User %s not found in users table. Returning empty tenant list.\n", userID)
			return []TenantResponse{}, nil
		}
		return nil, fmt.Errorf("failed to fetch user tenant info: %w", err)
	}

	// 2. Fetch all tenants (with full hierarchy)
	// We pass the explicit tenantID to filter efficiently at the DB level
	var targetTenantID *string
	if tenantID.Valid && tenantID.String != "" {
		s := tenantID.String
		targetTenantID = &s
	}

	allTenants, err := h.getAllTenantsInternal(ctx, targetTenantID)
	if err != nil {
		return nil, err
	}

	// 3. Filter results (Double check: if we passed filter, DB should have filtered, but safety first)
	var accessible []TenantResponse

	if targetTenantID != nil {
		// User is bound to a single tenant; only return that one
		targetID := *targetTenantID
		for _, t := range allTenants {
			if t.ID == targetID {
				accessible = append(accessible, t)
			}
		}
		// If user is bound to a tenant but it's not in the list (e.g. inactive), return empty
		return accessible, nil
	}

	// 4. Default: No access if no tenant_id found
	// We explicitly DO NOT fall back to returning all tenants.
	fmt.Printf("[DEBUG] User %s has no tenant_id assigned. Returning empty list.\n", userID)
	return []TenantResponse{}, nil
}
