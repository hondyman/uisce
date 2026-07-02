package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/security"
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

	// Admin tenant access mapping endpoints
	r.Route("/admin/tenant-access", func(r chi.Router) {
		r.Get("/", h.listTenantAccessMappings)
		r.Post("/", h.createTenantAccessMapping)
		r.Get("/tenants", h.listAllTenants)
		r.Route("/{id}", func(r chi.Router) {
			r.Put("/", h.updateTenantAccessMapping)
			r.Delete("/", h.deleteTenantAccessMapping)
		})
	})
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
			http.Error(w, "Panic", http.StatusInternalServerError)
		}
	}()

	actor, ok := h.isPlatformOperator(r)
	fmt.Printf("[DEBUG] listAccessibleTenants: UserID=%s IsPlatform=%v\n", actor.UserID, ok)

	if ok {
		h.listAllTenants(w, r)
		return
	}

	// For non-operators, filter by tenant assignments
	tenants, err := h.getTenantsByUser(r, actor.UserID)
	if err != nil {
		fmt.Printf("[DEBUG] getTenantsByUser error: %v\n", err)
		http.Error(w, "Failed to fetch accessible tenants: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[DEBUG] listAccessibleTenants found %d tenants for user %s\n", len(tenants), actor.UserID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

// listAllTenants returns tenants the caller is authorized to see.
// Global admins see all tenants; tenant managers see only their assigned tenant(s).
func (h *TenantAccessHandlers) listAllTenants(w http.ResponseWriter, r *http.Request) {
	actor, isPlatform := h.isPlatformOperator(r)

	var tenants []TenantResponse
	var err2 error

	if isPlatform {
		tenants, err2 = h.getAllTenantsInternal(r.Context(), nil)
	} else {
		// Verify if they are tenant managers (tenant_admin or professional_services)
		isManager := false
		var tenantIDs []string

		if actor.UserID != "" && hasAnyRole(actor.Roles, []string{"tenant_admin", "professional_services"}) {
			isManager = true
			tenantIDs = actor.TenantIDs
		} else {
			// Fallback to headers for legacy tests
			userRole := r.Header.Get("X-User-Role")
			if userRole == "tenant_admin" || userRole == "professional_services" {
				isManager = true
				tenantID := r.Header.Get("X-Tenant-ID")
				if tenantID != "" {
					tenantIDs = []string{tenantID}
				}
			}
		}

		if !isManager {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		tenants = []TenantResponse{}
		for _, tid := range tenantIDs {
			t, err := h.getAllTenantsInternal(r.Context(), &tid)
			if err != nil {
				http.Error(w, "Failed to query tenants: "+err.Error(), http.StatusInternalServerError)
				return
			}
			tenants = append(tenants, t...)
		}
	}

	if err2 != nil {
		http.Error(w, "Failed to query tenants: "+err2.Error(), http.StatusInternalServerError)
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

// getTenantsByUser returns tenants accessible to a specific user.
// Lease-based operators (helpdesk / professional services) have their tenant
// context set by the auth middleware in X-Tenant-ID after lease verification.
// That explicit context takes precedence over any persistent users.tenant_id
// binding so that a support operator can never accidentally see multiple tenants.
func (h *TenantAccessHandlers) getTenantsByUser(r *http.Request, userID string) ([]TenantResponse, error) {
	ctx := r.Context()

	// Defensive: requests without a resolved user should not crash the DB query.
	if strings.TrimSpace(userID) == "" {
		fmt.Printf("[DEBUG] getTenantsByUser called with empty userID. Returning empty tenant list.\n")
		return []TenantResponse{}, nil
	}

	// 1. Prefer the explicit tenant context from the auth middleware. This is the
	// authoritative tenant for lease-scoped operators and for any request where a
	// tenant has been explicitly selected.
	if explicitTenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); explicitTenantID != "" {
		return h.getAllTenantsInternal(ctx, &explicitTenantID)
	}

	// 2. Check for lease-scoped support operator mappings.
	userRole := r.Header.Get("X-User-Role")
	userEmail := r.Header.Get("X-User-Email")
	if (userRole == "professional_services" || userRole == "helpdesk") && userEmail != "" {
		rows, err := h.DB.QueryContext(ctx, `
			SELECT target_tenant_id 
			FROM security.staff_tenant_assignments 
			WHERE operator_email = $1 AND expires_at > NOW()
		`, userEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch staff tenant assignments: %w", err)
		}
		defer rows.Close()

		var tenantIDs []string
		for rows.Next() {
			var tid string
			if err := rows.Scan(&tid); err != nil {
				return nil, err
			}
			tenantIDs = append(tenantIDs, tid)
		}

		if len(tenantIDs) == 0 {
			return []TenantResponse{}, nil
		}

		var accessibleTenants []TenantResponse
		for _, tid := range tenantIDs {
			t, err := h.getAllTenantsInternal(ctx, &tid)
			if err != nil {
				return nil, err
			}
			accessibleTenants = append(accessibleTenants, t...)
		}
		return accessibleTenants, nil
	}

	// 2b. Check public.user_tenant assignments (preferred many-to-many mapping).
	rows, err := h.DB.QueryContext(ctx, `
		SELECT tenant_id
		FROM public.user_tenant
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user tenant assignments: %w", err)
	}
	defer rows.Close()

	var tenantIDs []string
	for rows.Next() {
		var tid *string
		if err := rows.Scan(&tid); err != nil {
			return nil, err
		}
		if tid != nil && *tid != "" {
			tenantIDs = append(tenantIDs, *tid)
		}
	}

	if len(tenantIDs) > 0 {
		var accessibleTenants []TenantResponse
		for _, tid := range tenantIDs {
			t, err := h.getAllTenantsInternal(ctx, &tid)
			if err != nil {
				return nil, err
			}
			accessibleTenants = append(accessibleTenants, t...)
		}
		return accessibleTenants, nil
	}

	// 3. Fall back to the persistent tenant binding in the users table.
	var tenantID sql.NullString
	err = h.DB.QueryRowContext(ctx, "SELECT tenant_id FROM users WHERE id = $1", userID).Scan(&tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			// User has not been provisioned in the platform yet; treat as no accessible tenants.
			fmt.Printf("[DEBUG] User %s not found in users table. Returning empty tenant list.\n", userID)
			return []TenantResponse{}, nil
		}
		return nil, fmt.Errorf("failed to fetch user tenant info: %w", err)
	}

	// 4. Fetch tenants (with full hierarchy) filtered to the bound tenant.
	var targetTenantID *string
	if tenantID.Valid && tenantID.String != "" {
		s := tenantID.String
		targetTenantID = &s
	}

	if targetTenantID == nil {
		// No explicit context and no persistent binding means no access.
		fmt.Printf("[DEBUG] User %s has no tenant_id assigned. Returning empty list.\n", userID)
		return []TenantResponse{}, nil
	}

	return h.getAllTenantsInternal(ctx, targetTenantID)
}

func (h *TenantAccessHandlers) listTenantAccessMappings(w http.ResponseWriter, r *http.Request) {
	actor, ok, err := h.requireTenantManager(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	query := `
		SELECT ut.user_id, ut.tenant_id, COALESCE(ut.access_role, 'viewer') as access_role,
		       COALESCE(u.email, '') as email, ut.created_at, ut.updated_at
		FROM public.user_tenant ut
		LEFT JOIN public.app_user u ON ut.user_id = u.id
	`
	args := []interface{}{}

	if !isGlobalAdmin(actor) {
		if len(actor.TenantIDs) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		placeholders := make([]string, len(actor.TenantIDs))
		for i, tid := range actor.TenantIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, tid)
		}
		query += ` WHERE ut.tenant_id IN (` + strings.Join(placeholders, ",") + `)`
	}

	query += ` ORDER BY ut.created_at DESC`

	rows, err := h.DB.QueryContext(ctx, query, args...)
	if err != nil {
		http.Error(w, "Failed to query tenant access mappings: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Mapping struct {
		ID         string    `json:"id"`
		UserID     string    `json:"user_id"`
		TenantID   string    `json:"tenant_id"`
		AccessRole string    `json:"access_role"`
		Email      string    `json:"email"`
		CreatedAt  time.Time `json:"created_at"`
		UpdatedAt  time.Time `json:"updated_at"`
	}

	mappings := []Mapping{}
	for rows.Next() {
		var m Mapping
		if err := rows.Scan(&m.UserID, &m.TenantID, &m.AccessRole, &m.Email, &m.CreatedAt, &m.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan tenant access mapping: "+err.Error(), http.StatusInternalServerError)
			return
		}
		m.ID = m.UserID + ":" + m.TenantID
		mappings = append(mappings, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mappings)
}

func (h *TenantAccessHandlers) createTenantAccessMapping(w http.ResponseWriter, r *http.Request) {
	actor, ok, err := h.requireTenantManager(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		UserID     string `json:"user_id"`
		TenantID   string `json:"tenant_id"`
		AccessRole string `json:"access_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.TenantID == "" {
		http.Error(w, "Missing user_id or tenant_id", http.StatusBadRequest)
		return
	}

	if !canManageTenantAccess(actor, req.TenantID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if req.AccessRole == "" {
		req.AccessRole = "viewer"
	}

	ctx := r.Context()
	query := `
		INSERT INTO public.user_tenant (user_id, tenant_id, access_role, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, tenant_id) DO UPDATE
		SET access_role = EXCLUDED.access_role, updated_at = NOW()
	`
	_, err = h.DB.ExecContext(ctx, query, req.UserID, req.TenantID, req.AccessRole)
	if err != nil {
		http.Error(w, "Failed to create tenant access mapping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *TenantAccessHandlers) deleteTenantAccessMapping(w http.ResponseWriter, r *http.Request) {
	actor, ok, err := h.requireTenantManager(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		http.Error(w, "Invalid mapping ID format. Expected user_id:tenant_id", http.StatusBadRequest)
		return
	}
	userID, tenantID := parts[0], parts[1]

	if !canManageTenantAccess(actor, tenantID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ctx := r.Context()
	query := `DELETE FROM public.user_tenant WHERE user_id = $1 AND tenant_id = $2`
	_, err = h.DB.ExecContext(ctx, query, userID, tenantID)
	if err != nil {
		http.Error(w, "Failed to delete tenant access mapping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TenantAccessHandlers) updateTenantAccessMapping(w http.ResponseWriter, r *http.Request) {
	actor, ok, err := h.requireTenantManager(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		http.Error(w, "Invalid mapping ID format. Expected user_id:tenant_id", http.StatusBadRequest)
		return
	}
	userID, tenantID := parts[0], parts[1]

	if !canManageTenantAccess(actor, tenantID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		AccessRole string `json:"access_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AccessRole == "" {
		http.Error(w, "Missing access_role", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := `UPDATE public.user_tenant SET access_role = $1, updated_at = NOW() WHERE user_id = $2 AND tenant_id = $3`
	_, err = h.DB.ExecContext(ctx, query, req.AccessRole, userID, tenantID)
	if err != nil {
		http.Error(w, "Failed to update tenant access mapping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ============================================================================
// Authorization helpers for tenant access management
// ============================================================================

func (h *TenantAccessHandlers) requireGlobalAdmin(r *http.Request) (security.AuthInfo, bool) {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		return security.AuthInfo{}, false
	}
	return actor, isGlobalAdmin(actor)
}

func (h *TenantAccessHandlers) requireTenantManager(r *http.Request) (security.AuthInfo, bool, error) {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		return security.AuthInfo{}, false, errors.New("unauthorized")
	}
	if isGlobalAdmin(actor) {
		return actor, true, nil
	}
	if hasAnyRole(actor.Roles, []string{"tenant_admin", "professional_services"}) {
		return actor, true, nil
	}
	return security.AuthInfo{}, false, nil
}

func canManageTenantAccess(actor security.AuthInfo, tenantID string) bool {
	if isGlobalAdmin(actor) {
		return true
	}
	return tenantAllowed(actor.TenantIDs, tenantID)
}

func isGlobalAdmin(actor security.AuthInfo) bool {
	if actor.IsGlobalAdmin {
		return true
	}
	return hasAnyRole(actor.Roles, []string{"global_admin", "global_ops", "core_admin"})
}

func hasAnyRole(roles []string, targets []string) bool {
	for _, r := range roles {
		for _, t := range targets {
			if strings.EqualFold(strings.TrimSpace(r), t) {
				return true
			}
		}
	}
	return false
}

func tenantAllowed(allowed []string, tenantID string) bool {
	tid := strings.TrimSpace(tenantID)
	for _, candidate := range allowed {
		if strings.TrimSpace(candidate) == tid {
			return true
		}
	}
	return false
}

// Recognise the federated IdP group by name (defence-in-depth so the platform
// operator status is computed correctly even if the operator_role claim
// derivation in ValidateToken ever drops a group). Path form ("/Uisce-Global-Admins")
// and leaf form ("Uisce-Global-Admins") are both accepted.
func idpGroupGrantsPlatformOperator(groups []string) bool {
	for _, g := range groups {
		leaf := g
		if idx := strings.LastIndex(g, "/"); idx >= 0 {
			leaf = g[idx+1:]
		}
		lower := strings.ToLower(strings.TrimSpace(leaf))
		if lower == "uisce-global-admins" || lower == "uisce-global-admin" ||
			lower == "global-admin" || lower == "global_admin" {
			return true
		}
	}
	return false
}

func (h *TenantAccessHandlers) isPlatformOperator(r *http.Request) (security.AuthInfo, bool) {
	// 1. Try security.AuthInfoFromContext first (preferred path)
	if actor, ok := security.AuthInfoFromContext(r.Context()); ok && strings.TrimSpace(actor.UserID) != "" {
		if actor.IsGlobalAdmin || hasAnyRole(actor.Roles, []string{"global_admin", "global_ops", "core_admin"}) {
			return actor, true
		}
		// Defence-in-depth: even if the operator_role claim was not populated, a
		// federated IdP group membership still grants platform-operator status.
		if idpGroupGrantsPlatformOperator(actor.IDPGroups) {
			actor.IsGlobalAdmin = true
			return actor, true
		}
		// Also check userRole or permissions if any
		userRole := r.Header.Get("X-User-Role")
		isCoreAdmin := r.Header.Get("X-Is-Core-Admin") == "true"
		if isCoreAdmin ||
			userRole == "platform_operator" ||
			userRole == "admin" ||
			userRole == "global_admin" ||
			strings.Contains(r.Header.Get("X-User-Permissions"), "platform:operator") {
			return actor, true
		}
		return actor, false
	}

	// 2. Fall back to header sniffing for legacy callers / tests
	userRole := r.Header.Get("X-User-Role")
	userID := r.Header.Get("X-User-ID")
	isCoreAdmin := r.Header.Get("X-Is-Core-Admin") == "true"

	isPlatform := isCoreAdmin ||
		userRole == "platform_operator" ||
		userRole == "admin" ||
		userRole == "global_admin" ||
		strings.Contains(r.Header.Get("X-User-Permissions"), "platform:operator")

	if isPlatform && userID != "" {
		actor := security.AuthInfo{
			UserID:        userID,
			IsGlobalAdmin: true,
			Roles:         []string{userRole},
		}
		return actor, true
	}

	return security.AuthInfo{UserID: userID, Roles: []string{userRole}}, false
}
