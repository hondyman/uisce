package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/auth"
	"github.com/lib/pq"
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
//
// This endpoint is fail-closed: if we cannot identify the caller (no auth
// context) or a tenant user fails to supply an explicit X-Tenant-Id, we
// reject the request rather than returning a silent empty list.
func (h *TenantAccessHandlers) listAccessibleTenants(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Printf("PANIC in listAccessibleTenants: %v\n", rec)
			http.Error(w, "Panic", http.StatusInternalServerError)
		}
	}()

	authInfo, ok := security.AuthInfoFromContext(r.Context())
	if !ok || authInfo.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := authInfo.UserID
	isPlatformOperator := authInfo.IsGlobalAdmin

	fmt.Printf("[DEBUG] listAccessibleTenants: UserID=%s PlatformOp=%v\n", userID, isPlatformOperator)

	if isPlatformOperator {
		h.listAllTenants(w, r)
		return
	}

	tenantIDHeader := r.Header.Get("X-Tenant-Id")
	if tenantIDHeader == "" {
		http.Error(w, "Forbidden: tenant scope required", http.StatusForbidden)
		return
	}

	claims, err := jwtmiddleware.ValidateTokenFromRequest(r)
	if err != nil || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := jwtmiddleware.ValidateTenantAccess(claims, tenantIDHeader); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	var tenants []TenantResponse
	err = db.WithTenantTransaction(r.Context(), h.DB, tenantIDHeader, func(tx *sql.Tx) error {
		tenants, err = h.getTenantsByUser(r.Context(), userID, tx)
		return err
	})
	if err != nil {
		http.Error(w, "Failed to fetch accessible tenants: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[DEBUG] listAccessibleTenants found %d tenants for user %s\n", len(tenants), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

// listAllTenants returns all tenants with full hierarchy
func (h *TenantAccessHandlers) listAllTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.getAllTenantsInternal(r.Context(), nil, nil) // nil means fetch all
	if err != nil {
		http.Error(w, "Failed to query tenants: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

func (h *TenantAccessHandlers) getAllTenantsInternal(ctx context.Context, targetTenantID *string, tx *sql.Tx) ([]TenantResponse, error) {
	dbForQuery := func(query string, args ...interface{}) (*sql.Rows, error) {
		if tx != nil {
			return tx.QueryContext(ctx, query, args...)
		}
		return h.DB.QueryContext(ctx, query, args...)
	}

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

	tenantRows, err := dbForQuery(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer tenantRows.Close()

	var tenants []TenantResponse
	var tenantIDs []string
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
		tenantIDs = append(tenantIDs, t.ID)
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

	instanceRows, err := dbForQuery(instanceQuery, iArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query instances: %w", err)
	}
	defer instanceRows.Close()

	instanceMap := make(map[string][]InstanceResponse)
	var instanceIDs []string
	for instanceRows.Next() {
		var i InstanceResponse
		if err := instanceRows.Scan(&i.ID, &i.DisplayName, &i.Name, &i.Description, &i.IsActive, &i.URL, &i.TenantID); err != nil {
			return nil, err
		}
		i.Products = []ProductResponse{}
		instanceMap[i.TenantID] = append(instanceMap[i.TenantID], i)
		instanceIDs = append(instanceIDs, i.ID)
	}

	// 3. Query products
	// tenant_product.datasource_id is the FK to tenant_instance.id (the schema
	// does not have a tenant_instance_id column on tenant_product).
	// Note: id, datasource_id, and alpha_product_id are NOT NULL on
	// tenant_product — COALESCE on those columns is unnecessary and breaks
	// UUID→text inference at plan time. ap.id / ap.product_name can be NULL
	// due to the LEFT JOIN.
	productQuery := `
		SELECT tp.id, COALESCE(tp.version, 1.0) AS version,
		       tp.datasource_id,
		       tp.alpha_product_id,
		       ap.id AS ap_id,
		       COALESCE(ap.product_name, '') AS product_name,
		       ap.product_code,
		       COALESCE(ap.is_active, true) AS ap_is_active
		FROM tenant_product tp
		LEFT JOIN alpha_product ap ON ap.id = tp.alpha_product_id
		WHERE 1=1
	`
	var pArgs []interface{}
	if targetTenantID != nil {
		// Filter at DB level by joining back to tenant_instance via datasource_id.
		productQuery = `
			SELECT tp.id, COALESCE(tp.version, 1.0) AS version,
			       tp.datasource_id,
			       tp.alpha_product_id,
			       ap.id AS ap_id,
			       COALESCE(ap.product_name, '') AS product_name,
			       ap.product_code,
			       COALESCE(ap.is_active, true) AS ap_is_active
			FROM tenant_product tp
			JOIN tenant_instance ti ON tp.datasource_id = ti.id
			LEFT JOIN alpha_product ap ON ap.id = tp.alpha_product_id
			WHERE ti.tenant_id = $1
		`
		pArgs = append(pArgs, *targetTenantID)
	} else if len(instanceIDs) > 0 {
		// When fetching all, filter by the instances we found.
		productQuery += ` AND tp.datasource_id = ANY($1)`
		pArgs = append(pArgs, pq.Array(instanceIDs))
	} else {
		// No instances, so no products.
		productQuery += ` AND 1=0`
	}
	productQuery += " ORDER BY ap.product_name"

	productRows, err := dbForQuery(productQuery, pArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer productRows.Close()

	productMap := make(map[string][]ProductResponse)
	for productRows.Next() {
		var p ProductResponse
		var ap AlphaProductInfo
		var version sql.NullFloat64
		var alphaProductID sql.NullString
		var apID sql.NullString
		var apProductCode sql.NullString

		if err := productRows.Scan(&p.ID, &version, &p.TenantInstanceID, &alphaProductID,
			&apID, &ap.ProductName, &apProductCode, &ap.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan product row: %w", err)
		}
		if version.Valid {
			p.Version = version.Float64
		}
		if alphaProductID.Valid {
			p.AlphaProductID = alphaProductID.String
		}
		if apID.Valid {
			ap.ID = apID.String
		}
		if apProductCode.Valid {
			ap.ProductCode = &apProductCode.String
		}
		p.AlphaProduct = &ap
		p.Datasources = []DatasourceResponse{}
		productMap[p.TenantInstanceID] = append(productMap[p.TenantInstanceID], p)
	}

	// 4. Query datasources
	// tenant_product_datasource.datasource_id is the FK to alpha_datasource.id;
	// joining against it is what populates the alpha_datasource JSON block.
	// Note: tpd.id and tpd.tenant_product_id are NOT NULL uuids — do not
	// COALESCE those (breaks type inference). The LEFT JOIN to alpha_datasource
	// can produce NULL on every ads.* column, so we scan those via sql.NullString.
	dsQuery := `
		SELECT tpd.id, COALESCE(tpd.is_active, true) AS is_active,
		       COALESCE(tpd.source_name, '') AS source_name, tpd.tenant_product_id,
		       ads.id AS ads_id,
		       COALESCE(ads.datasource_name, '') AS ds_name,
		       COALESCE(ads.datasource_type, '') AS ds_type,
		       COALESCE(ads.datasource_code, '') AS ds_code
		FROM tenant_product_datasource tpd
		LEFT JOIN alpha_datasource ads ON ads.id = tpd.datasource_id
		WHERE 1=1
	`
	var dArgs []interface{}
	if targetTenantID != nil {
		dsQuery += ` AND tpd.tenant_id = $1`
		dArgs = append(dArgs, *targetTenantID)
	} else if len(instanceIDs) > 0 {
		dsQuery += ` AND tpd.tenant_product_id IN
		             (SELECT id FROM tenant_product WHERE datasource_id = ANY($1))`
		dArgs = append(dArgs, pq.Array(instanceIDs))
	} else {
		dsQuery += ` AND 1=0`
	}
	dsQuery += " ORDER BY tpd.source_name"

	dsRows, err := dbForQuery(dsQuery, dArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query datasources: %w", err)
	}
	defer dsRows.Close()

	dsMap := make(map[string][]DatasourceResponse)
	for dsRows.Next() {
		var ds DatasourceResponse
		var ads AlphaDatasourceInfo
		var productID string
		var adsID sql.NullString
		if err := dsRows.Scan(&ds.ID, &ds.IsActive, &ds.SourceName, &productID,
			&adsID, &ads.DatasourceName, &ads.DatasourceType, &ads.DatasourceCode); err != nil {
			return nil, fmt.Errorf("failed to scan datasource row: %w", err)
		}
		if adsID.Valid {
			ads.ID = adsID.String
		}
		ds.AlphaDatasource = &ads
		dsMap[productID] = append(dsMap[productID], ds)
	}

	// Assemble the hierarchy
	for i := range tenants {
		if instances, ok := instanceMap[tenants[i].ID]; ok {
			for j := range instances {
				if products, ok := productMap[instances[j].ID]; ok {
					for k := range products {
						if datasources, ok := dsMap[products[k].ID]; ok {
							products[k].Datasources = datasources
						}
					}
					instances[j].Products = products
				}
			}
			tenants[i].Instances = instances
		}
	}

	return tenants, nil
}

func (h *TenantAccessHandlers) getTenantsByUser(ctx context.Context, userID string, tx *sql.Tx) ([]TenantResponse, error) {
	dbForQueryRow := func(query string, args ...interface{}) *sql.Row {
		if tx != nil {
			return tx.QueryRowContext(ctx, query, args...)
		}
		return h.DB.QueryRowContext(ctx, query, args...)
	}

	var tenantID sql.NullString
	err := dbForQueryRow("SELECT tenant_id FROM users WHERE id = $1", userID).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user tenant info: %w", err)
	}

	var targetTenantID *string
	if tenantID.Valid && tenantID.String != "" {
		s := tenantID.String
		targetTenantID = &s
	}

	allTenants, err := h.getAllTenantsInternal(ctx, targetTenantID, tx)
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
