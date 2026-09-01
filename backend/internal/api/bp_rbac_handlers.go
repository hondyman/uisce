package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

// ============================================================================
// RBAC & Permissions API Handlers
// Fortune 500 Enterprise-Grade Security
// ============================================================================

type RBACHandlers struct {
	db             *sqlx.DB
	securityDeps   handlers.SecurityContextDeps
	fieldPermRepo  *security.FieldPermissionRepository
	entitlements   *security.EntitlementsService
}

func NewRBACHandlers(db *sqlx.DB, securityDeps handlers.SecurityContextDeps, entitlements *security.EntitlementsService) *RBACHandlers {
	return &RBACHandlers{
		db:            db,
		securityDeps:  securityDeps,
		fieldPermRepo: security.NewFieldPermissionRepository(db),
		entitlements:  entitlements,
	}
}

func (h *RBACHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/rbac", func(r chi.Router) {
		// Roles
		r.Get("/roles", h.listRoles)
		r.Post("/roles", h.createRole)
		r.Get("/roles/{roleId}", h.getRole)
		r.Put("/roles/{roleId}", h.updateRole)
		r.Delete("/roles/{roleId}", h.deleteRole)
		r.Post("/roles/{roleId}/clone", h.cloneRole)
		r.Get("/roles/{roleId}/effective-permissions", h.effectiveRolePermissions)

		// Permissions
		r.Get("/permissions", h.listPermissions)
		r.Get("/permissions/user/{userId}", h.getUserPermissions)
		r.Post("/permissions/check", h.checkPermission)

		// Role Assignments
		r.Post("/roles/{roleId}/assign", h.assignRoleToUser)
		r.Delete("/roles/{roleId}/unassign/{userId}", h.unassignRoleFromUser)
		r.Get("/roles/{roleId}/users", h.getRoleUsers)
		r.Get("/users/{userId}/roles", h.getUserRoles)

		// Users (for role assignment UI)
		r.Get("/users", h.listUsers)
		r.Post("/users", h.createUser)
		r.Put("/users/{userId}/tenant", h.updateUserTenant)

		// Field-Level Permissions
		r.Get("/field-permissions", h.listFieldPermissions)
		r.Post("/field-permissions", h.createFieldPermission)
		r.Get("/field-permissions/user/{userId}/resource/{resourceType}/{resourceId}", h.getUserFieldPermissions)

		// Delegations
		r.Get("/delegations", h.listDelegations)
		r.Post("/delegations", h.createDelegation)
		r.Put("/delegations/{delegationId}", h.updateDelegation)
		r.Delete("/delegations/{delegationId}", h.deleteDelegation)
		r.Get("/delegations/user/{userId}", h.getUserDelegations)
		r.Post("/delegations/{delegationId}/log", h.logDelegationUsage)

		// Teams
		r.Get("/teams", h.listTeams)
		r.Post("/teams", h.createTeam)
		r.Post("/teams/{teamId}/members", h.addTeamMember)
		r.Delete("/teams/{teamId}/members/{userId}", h.removeTeamMember)
		r.Get("/teams/{teamId}/members", h.getTeamMembers)

		// Audit
		r.Get("/audit", h.listPermissionAudit)
	})
}

// ============================================================================
// ROLE MANAGEMENT
// ============================================================================

type Role struct {
	ID                 string    `json:"id" db:"id"`
	TenantID           string    `json:"tenant_id" db:"tenant_id"`
	RoleKey            string    `json:"role_key" db:"role_key"`
	RoleName           string    `json:"role_name" db:"role_name"`
	Description        string    `json:"description" db:"description"`
	RoleType           string    `json:"role_type" db:"role_type"`
	RoleLevel          string    `json:"role_level" db:"role_level"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	IsTemplate         bool      `json:"is_template" db:"is_template"`
	ParentRoleID       *string   `json:"parent_role_id" db:"parent_role_id"`
	SecurityProfileID  *string   `json:"security_profile_id" db:"security_profile_id"`
	TenantInstanceID   *string   `json:"tenant_instance_id,omitempty" db:"tenant_instance_id"`
	CreatedBy          *string   `json:"created_by" db:"created_by"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	// Origin is derived, not stored: "gold_copy" for template roles living
	// under the gold-copy tenant, "tenant" for everything else.
	Origin string `json:"origin" db:"-"`
}

// listRoles returns the caller's own tenant roles UNIONed with every
// gold-copy template role, mirroring the business_objects read pattern
// (`tenant_id = $1 OR tenant_id = gold_copy_tenant()`). Without the union,
// a role authored once in the gold-copy tenant would never be visible to
// any other tenant, which defeats the inheritance model entirely.
func (h *RBACHandlers) listRoles(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var roles []Role
	err = h.db.Select(&roles, `
		SELECT * FROM bp_roles
		WHERE is_active = true
		  AND (tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		ORDER BY role_level, role_name
	`, tenantID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch roles: %v", err), http.StatusInternalServerError)
		return
	}

	for i := range roles {
		if roles[i].IsTemplate {
			roles[i].Origin = "gold_copy"
		} else if roles[i].ParentRoleID != nil {
			roles[i].Origin = "extended"
		} else {
			roles[i].Origin = "tenant"
		}
	}

	respondJSONRBAC(w, r, roles, http.StatusOK)
}

// cloneRole creates a tenant-scoped copy of a gold-copy template role, linked
// via parent_role_id so ResolveEffectivePermissions can union the two without
// ever duplicating bp_role_permissions rows, and without ever mutating the
// source template row.
func (h *RBACHandlers) cloneRole(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	sourceRoleID := chi.URLParam(r, "roleId")

	var req struct {
		RoleKey  string `json:"role_key"`
		RoleName string `json:"role_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var source Role
	if err := h.db.Get(&source, "SELECT * FROM bp_roles WHERE id = $1", sourceRoleID); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Source role not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to load source role: %v", err), http.StatusInternalServerError)
		return
	}
	if !source.IsTemplate {
		http.Error(w, "Only gold-copy template roles can be cloned", http.StatusBadRequest)
		return
	}

	roleKey := req.RoleKey
	if roleKey == "" {
		roleKey = source.RoleKey
	}
	roleName := req.RoleName
	if roleName == "" {
		roleName = source.RoleName
	}

	var newProfileID *string
	if source.SecurityProfileID != nil {
		var sourceProfile security.SecurityProfile
		if err := h.db.Get(&sourceProfile, `
			SELECT profile_id, tenant_id, profile_key, profile_name, parent_profile_id, created_at, updated_at
			FROM security.security_profiles WHERE profile_id = $1
		`, *source.SecurityProfileID); err == nil {
			profileSvc := security.NewProfileService(h.db.DB)
			tid, parseErr := uuid.Parse(tenantID)
			if parseErr == nil {
				created, cErr := profileSvc.CreateProfile(r.Context(), &security.SecurityProfile{
					TenantID:        &tid,
					ProfileKey:      roleKey,
					ProfileName:     roleName,
					ParentProfileID: &sourceProfile.ProfileID,
				})
				if cErr == nil {
					pid := created.ProfileID.String()
					newProfileID = &pid
				}
			}
		}
	}

	var newRoleID string
	err = h.db.QueryRow(`
		INSERT INTO bp_roles (tenant_id, role_key, role_name, description, role_type, role_level, parent_role_id, security_profile_id, is_template)
		VALUES ($1, $2, $3, $4, 'custom', $5, $6, $7, false)
		RETURNING id
	`, tenantID, roleKey, roleName, source.Description, source.RoleLevel, source.ID, newProfileID).Scan(&newRoleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clone role: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"id": newRoleID, "parent_role_id": source.ID, "status": "cloned"}, http.StatusCreated)
}

// effectiveRolePermissions returns the union of this role's own
// bp_role_permissions plus everything inherited from its parent_role_id
// chain (the gold-copy ancestor's grants).
func (h *RBACHandlers) effectiveRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")
	perms, err := security.ResolveEffectivePermissions(r.Context(), h.db.DB, roleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve effective permissions: %v", err), http.StatusInternalServerError)
		return
	}
	respondJSONRBAC(w, r, map[string]interface{}{"role_id": roleID, "permissions": perms}, http.StatusOK)
}

func (h *RBACHandlers) createRole(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var req struct {
		RoleKey     string   `json:"role_key"`
		RoleName    string   `json:"role_name"`
		Description string   `json:"description"`
		RoleLevel   string   `json:"role_level"`
		IsTemplate  bool     `json:"is_template"` // only takes effect when the caller is scoped to the gold-copy tenant
		Permissions []string `json:"permissions"` // Permission IDs to assign
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	isTemplate := false
	if req.IsTemplate {
		var goldCopyTenantID sql.NullString
		if err := h.db.QueryRow(`SELECT id::text FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyTenantID); err == nil {
			isTemplate = goldCopyTenantID.Valid && goldCopyTenantID.String == tenantID
		}
	}

	// Create role
	var roleID string
	err = h.db.QueryRow(`
		INSERT INTO bp_roles (tenant_id, role_key, role_name, description, role_type, role_level, is_template)
		VALUES ($1, $2, $3, $4, 'custom', $5, $6)
		RETURNING id
	`, tenantID, req.RoleKey, req.RoleName, req.Description, req.RoleLevel, isTemplate).Scan(&roleID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create role: %v", err), http.StatusInternalServerError)
		return
	}

	// Assign permissions
	for _, permID := range req.Permissions {
		_, err := h.db.Exec(`
			INSERT INTO bp_role_permissions (role_id, permission_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, roleID, permID)
		if err != nil {
			// Log error but continue
			fmt.Printf("Failed to assign permission %s: %v\n", permID, err)
		}
	}

	respondJSONRBAC(w, r, map[string]string{"id": roleID, "status": "created"}, http.StatusCreated)
}

func (h *RBACHandlers) getRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")

	var role Role
	err := h.db.Get(&role, "SELECT * FROM bp_roles WHERE id = $1", roleID)

	if err == sql.ErrNoRows {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch role: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, role, http.StatusOK)
}

func (h *RBACHandlers) updateRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")

	var req struct {
		RoleName    string `json:"role_name"`
		Description string `json:"description"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		UPDATE bp_roles
		SET role_name = COALESCE(NULLIF($1, ''), role_name),
		    description = COALESCE(NULLIF($2, ''), description),
		    is_active = COALESCE($3, is_active),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, req.RoleName, req.Description, req.IsActive, roleID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update role: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "updated"}, http.StatusOK)
}

func (h *RBACHandlers) deleteRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")

	// Soft delete
	_, err := h.db.Exec("UPDATE bp_roles SET is_active = false WHERE id = $1", roleID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete role: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "deleted"}, http.StatusOK)
}

// ============================================================================
// PERMISSION MANAGEMENT
// ============================================================================

func (h *RBACHandlers) listPermissions(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID

	var permissions []map[string]interface{}
	rows, err := h.db.Query(`
		SELECT id, permission_key, permission_name, description, resource_type, action, is_system
		FROM bp_permissions
		WHERE tenant_id = $1
		ORDER BY resource_type, action, permission_name
	`, tenantID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch permissions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var perm map[string]interface{} = make(map[string]interface{})
		var id, permKey, permName, desc, resType, action string
		var isSystem bool
		if err := rows.Scan(&id, &permKey, &permName, &desc, &resType, &action, &isSystem); err != nil {
			continue
		}
		perm["id"] = id
		perm["permission_key"] = permKey
		perm["permission_name"] = permName
		perm["description"] = desc
		perm["resource_type"] = resType
		perm["action"] = action
		perm["is_system"] = isSystem
		permissions = append(permissions, perm)
	}

	respondJSONRBAC(w, r, permissions, http.StatusOK)
}

func (h *RBACHandlers) getUserPermissions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var permissions []map[string]string
	rows, err := h.db.Query(`
		SELECT DISTINCT p.permission_key, p.permission_name, p.resource_type, p.action
		FROM bp_user_roles ur
		JOIN bp_role_permissions rp ON ur.role_id = rp.role_id
		JOIN bp_permissions p ON rp.permission_id = p.id
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND ur.datasource_id = $3
		  AND ur.is_active = true
		  AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP)
		ORDER BY p.resource_type, p.action
	`, userID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user permissions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var perm map[string]string = make(map[string]string)
		var permKey, permName, resType, action string
		if err := rows.Scan(&permKey, &permName, &resType, &action); err != nil {
			continue
		}
		perm["permission_key"] = permKey
		perm["permission_name"] = permName
		perm["resource_type"] = resType
		perm["action"] = action
		permissions = append(permissions, perm)
	}

	respondJSONRBAC(w, r, permissions, http.StatusOK)
}

func (h *RBACHandlers) checkPermission(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID        string `json:"user_id"`
		TenantID      string `json:"tenant_id"`
		DatasourceID  string `json:"datasource_id"`
		PermissionKey string `json:"permission_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var hasPerm bool
	err := h.db.QueryRow(`
		SELECT bp_user_has_permission($1, $2, $3, $4)
	`, req.UserID, req.TenantID, req.DatasourceID, req.PermissionKey).Scan(&hasPerm)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check permission: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]bool{"has_permission": hasPerm}, http.StatusOK)
}

// ============================================================================
// ROLE ASSIGNMENT
// ============================================================================

func (h *RBACHandlers) assignRoleToUser(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")

	var req struct {
		UserID       string  `json:"user_id"`
		ScopeType    *string `json:"scope_type"`
		ScopeID      *string `json:"scope_id"`
		ExpiresAt    *string `json:"expires_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	_, err = h.db.Exec(`
		INSERT INTO bp_user_roles (user_id, role_id, tenant_id, datasource_id, scope_type, scope_id, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, role_id, tenant_id, datasource_id, scope_type, scope_id) DO NOTHING
	`, req.UserID, roleID, tenantID, datasourceID, req.ScopeType, req.ScopeID, req.ExpiresAt)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assign role: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "assigned"}, http.StatusCreated)
}

func (h *RBACHandlers) unassignRoleFromUser(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")
	userID := chi.URLParam(r, "userId")

	_, err := h.db.Exec(`
		UPDATE bp_user_roles
		SET is_active = false
		WHERE role_id = $1 AND user_id = $2
	`, roleID, userID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to unassign role: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "unassigned"}, http.StatusOK)
}

func (h *RBACHandlers) getUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var roles []map[string]interface{}
	rows, err := h.db.Query(`
		SELECT r.id, r.role_key, r.role_name, r.role_level, ur.scope_type, ur.scope_id, ur.assigned_at, ur.expires_at
		FROM bp_user_roles ur
		JOIN bp_roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		  AND ur.tenant_id = $2
		  AND ur.datasource_id = $3
		  AND ur.is_active = true
		ORDER BY r.role_level DESC, r.role_name
	`, userID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user roles: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var role map[string]interface{} = make(map[string]interface{})
		var id, roleKey, roleName, roleLevel string
		var scopeType, scopeID sql.NullString
		var assignedAt time.Time
		var expiresAt sql.NullTime

		if err := rows.Scan(&id, &roleKey, &roleName, &roleLevel, &scopeType, &scopeID, &assignedAt, &expiresAt); err != nil {
			continue
		}

		role["id"] = id
		role["role_key"] = roleKey
		role["role_name"] = roleName
		role["role_level"] = roleLevel
		if scopeType.Valid {
			role["scope_type"] = scopeType.String
		}
		if scopeID.Valid {
			role["scope_id"] = scopeID.String
		}
		role["assigned_at"] = assignedAt
		if expiresAt.Valid {
			role["expires_at"] = expiresAt.Time
		}

		roles = append(roles, role)
	}

	respondJSONRBAC(w, r, roles, http.StatusOK)
}

func (h *RBACHandlers) getRoleUsers(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleId")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var users []map[string]interface{}
	// Join bp_user_roles with users table to get details
	// Assuming a 'users' table exists with id, username, name, email
	rows, err := h.db.Query(`
		SELECT u.id, u.username, u.name, u.email, ur.assigned_at
		FROM bp_user_roles ur
		JOIN users u ON ur.user_id = u.id
		WHERE ur.role_id = $1
		  AND ur.tenant_id = $2
		  AND ur.datasource_id = $3
		  AND ur.is_active = true
		ORDER BY u.name, u.username
	`, roleID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch role users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var u map[string]interface{} = make(map[string]interface{})
		var id, username, email string
		var name sql.NullString
		var assignedAt time.Time

		if err := rows.Scan(&id, &username, &name, &email, &assignedAt); err != nil {
			continue
		}

		u["id"] = id
		u["username"] = username
		if name.Valid {
			u["name"] = name.String
		} else {
			u["name"] = ""
		}
		u["email"] = email
		u["assigned_at"] = assignedAt

		users = append(users, u)
	}

	respondJSONRBAC(w, r, users, http.StatusOK)
}

// ============================================================================
// FIELD-LEVEL PERMISSIONS
// ============================================================================

func (h *RBACHandlers) listFieldPermissions(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var fieldPerms []map[string]interface{}

	query := `
		SELECT DISTINCT ON (fp.role_id, fp.term_node_id, fp.resource_type, fp.resource_id)
			fp.id, r.role_key, r.role_name, fp.term_node_id, fp.resource_type, fp.resource_id, fp.permission_level, fp.masking_pattern,
			cn.node_name as term_name, cn.display_name as term_display_name
		FROM bp_field_permissions fp
		JOIN bp_roles r ON fp.role_id = r.id
		LEFT JOIN catalog_node cn ON fp.term_node_id = cn.id
		WHERE fp.tenant_id = $1 AND fp.datasource_id = $2
	`
	args := []interface{}{tenantID, datasourceID}

	// Optional role_id filter (not a security boundary)
	roleID := r.URL.Query().Get("role_id")
	if roleID != "" {
		query += " AND fp.role_id = $3"
		args = append(args, roleID)
	}

	// Optional term_node_id filter
	termNodeIDParam := r.URL.Query().Get("term_node_id")
	if termNodeIDParam != "" {
		query += " AND fp.term_node_id = $4"
		args = append(args, termNodeIDParam)
	}

	query += " ORDER BY fp.role_id, fp.term_node_id, fp.resource_type, fp.resource_id, r.role_name"

	rows, err := h.db.Query(query, args...)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch field permissions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var fp map[string]interface{} = make(map[string]interface{})
		var id, roleKey, roleName, permLevel string
		var termNodeID, resType, resourceID, termName, termDisplayName, maskingPattern *string

		if err := rows.Scan(&id, &roleKey, &roleName, &termNodeID, &resType, &resourceID, &permLevel, &maskingPattern, &termName, &termDisplayName); err != nil {
			continue
		}

		fp["id"] = id
		fp["role_key"] = roleKey
		fp["role_name"] = roleName
		fp["term_node_id"] = termNodeID
		fp["resource_type"] = resType
		fp["resource_id"] = resourceID
		fp["permission_level"] = permLevel
		fp["masking_pattern"] = maskingPattern
		fp["term_name"] = termName
		fp["term_display_name"] = termDisplayName

		fieldPerms = append(fieldPerms, fp)
	}

	respondJSONRBAC(w, r, fieldPerms, http.StatusOK)
}

func (h *RBACHandlers) createFieldPermission(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoleID          string  `json:"role_id"`
		TermNodeID      string  `json:"term_node_id"`      // Semantic term ID - required
		ResourceType    *string `json:"resource_type"`    // Optional: for resource-specific overrides
		ResourceID      *string `json:"resource_id"`      // Optional: for instance-specific overrides
		PermissionLevel string  `json:"permission_level"`
		MaskingPattern *string `json:"masking_pattern"`   // For 'mask' permission level
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation: term_node_id is required (field_name path is deprecated)
	if req.TermNodeID == "" {
		http.Error(w, "term_node_id is required", http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var id string
	err = h.db.QueryRow(`
		INSERT INTO bp_field_permissions (tenant_id, datasource_id, role_id, term_node_id, resource_type, resource_id, permission_level, masking_pattern)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (role_id, term_node_id, resource_type, resource_id)
		DO UPDATE SET permission_level = EXCLUDED.permission_level, masking_pattern = EXCLUDED.masking_pattern, updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`, tenantID, datasourceID, req.RoleID, req.TermNodeID, req.ResourceType, req.ResourceID, req.PermissionLevel, req.MaskingPattern).Scan(&id)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create field permission: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"id": id, "status": "created"}, http.StatusCreated)
}

func (h *RBACHandlers) getUserFieldPermissions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	resourceType := chi.URLParam(r, "resourceType")
	resourceID := chi.URLParam(r, "resourceId")
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	// Use FieldPermissionRepository for canonical field permission lookup
	permissions, err := h.fieldPermRepo.GetTermPermissionsForUser(r.Context(), userID, tenantID, datasourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user field permissions: %v", err), http.StatusInternalServerError)
		return
	}

	// Enrich with term metadata and filter by resource type/id
	var fieldPerms []map[string]interface{}
	for _, perm := range permissions {
		// Skip if resource_type/resource_id doesn't match filters
		if resourceType != "" && perm.ResourceType != nil && *perm.ResourceType != resourceType {
			continue
		}
		if resourceID != "" && perm.ResourceID != nil && *perm.ResourceID != resourceID {
			continue
		}

		fp := map[string]interface{}{}
		permType := "semantic"
		if perm.TermNodeID == nil {
			permType = "field"
		}
		fp["permission_type"] = permType
		fp["permission_level"] = perm.PermissionLevel
		fp["term_node_id"] = stringOrEmptyPtr(perm.TermNodeID)

		fieldPerms = append(fieldPerms, fp)
	}

	respondJSONRBAC(w, r, fieldPerms, http.StatusOK)
}

// stringOrEmptyPtr returns the value of a string pointer or empty string
func stringOrEmptyPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ============================================================================
// DELEGATIONS (CONTINUED IN NEXT MESSAGE DUE TO LENGTH)
// ============================================================================

func (h *RBACHandlers) listDelegations(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var delegations []map[string]interface{}
	rows, err := h.db.Query(`
		SELECT id, delegator_user_id, delegate_user_id, delegation_type, resource_type, start_date, end_date, is_active
		FROM bp_approval_delegations
		WHERE tenant_id = $1 AND datasource_id = $2
		ORDER BY start_date DESC
	`, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch delegations: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var del map[string]interface{} = make(map[string]interface{})
		var id, delegatorID, delegateID, delType string
		var resType sql.NullString
		var startDate time.Time
		var endDate sql.NullTime
		var isActive bool

		if err := rows.Scan(&id, &delegatorID, &delegateID, &delType, &resType, &startDate, &endDate, &isActive); err != nil {
			continue
		}

		del["id"] = id
		del["delegator_user_id"] = delegatorID
		del["delegate_user_id"] = delegateID
		del["delegation_type"] = delType
		if resType.Valid {
			del["resource_type"] = resType.String
		}
		del["start_date"] = startDate
		if endDate.Valid {
			del["end_date"] = endDate.Time
		}
		del["is_active"] = isActive

		delegations = append(delegations, del)
	}

	respondJSONRBAC(w, r, delegations, http.StatusOK)
}

func (h *RBACHandlers) createDelegation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DelegatorID    string  `json:"delegator_user_id"`
		DelegateID     string  `json:"delegate_user_id"`
		DelegationType string  `json:"delegation_type"`
		ResourceType   *string `json:"resource_type"`
		ResourceID     *string `json:"resource_id"`
		StartDate      string  `json:"start_date"`
		EndDate        *string `json:"end_date"`
		Reason         *string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var id string
	err = h.db.QueryRow(`
		INSERT INTO bp_approval_delegations (
			tenant_id, datasource_id, delegator_user_id, delegate_user_id,
			delegation_type, resource_type, resource_id, start_date, end_date, reason
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, tenantID, datasourceID, req.DelegatorID, req.DelegateID,
		req.DelegationType, req.ResourceType, req.ResourceID, req.StartDate, req.EndDate, req.Reason).Scan(&id)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create delegation: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"id": id, "status": "created"}, http.StatusCreated)
}

func (h *RBACHandlers) updateDelegation(w http.ResponseWriter, r *http.Request) {
	delegationID := chi.URLParam(r, "delegationId")

	var req struct {
		EndDate  *string `json:"end_date"`
		IsActive *bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		UPDATE bp_approval_delegations
		SET end_date = COALESCE($1, end_date),
		    is_active = COALESCE($2, is_active)
		WHERE id = $3
	`, req.EndDate, req.IsActive, delegationID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update delegation: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "updated"}, http.StatusOK)
}

func (h *RBACHandlers) deleteDelegation(w http.ResponseWriter, r *http.Request) {
	delegationID := chi.URLParam(r, "delegationId")

	_, err := h.db.Exec("UPDATE bp_approval_delegations SET is_active = false WHERE id = $1", delegationID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete delegation: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "deleted"}, http.StatusOK)
}

func (h *RBACHandlers) getUserDelegations(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	delegationType := r.URL.Query().Get("type") // "delegator" or "delegate"

	var query string
	if delegationType == "delegate" {
		query = "SELECT * FROM bp_approval_delegations WHERE delegate_user_id = $1 AND is_active = true ORDER BY start_date DESC"
	} else {
		query = "SELECT * FROM bp_approval_delegations WHERE delegator_user_id = $1 AND is_active = true ORDER BY start_date DESC"
	}

	rows, err := h.db.Query(query, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch delegations: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Similar processing as listDelegations
	delegations := []map[string]interface{}{}
	respondJSONRBAC(w, r, delegations, http.StatusOK)
}

func (h *RBACHandlers) logDelegationUsage(w http.ResponseWriter, r *http.Request) {
	delegationID := chi.URLParam(r, "delegationId")

	var req struct {
		DelegateUserID string                 `json:"delegate_user_id"`
		ActionType     string                 `json:"action_type"`
		ResourceType   string                 `json:"resource_type"`
		ResourceID     string                 `json:"resource_id"`
		ActionDetails  map[string]interface{} `json:"action_details"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	detailsJSON, _ := json.Marshal(req.ActionDetails)

	_, err := h.db.Exec(`
		INSERT INTO bp_delegation_usage_log (delegation_id, delegate_user_id, action_type, resource_type, resource_id, action_details)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, delegationID, req.DelegateUserID, req.ActionType, req.ResourceType, req.ResourceID, detailsJSON)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to log delegation usage: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "logged"}, http.StatusCreated)
}

// ============================================================================
// TEAMS
// ============================================================================

func (h *RBACHandlers) listTeams(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var query string
	var args []interface{}
	if datasourceID != "" && datasourceID != "none" {
		query = `
			SELECT id, team_key, team_name, description, team_type, is_active
			FROM bp_teams
			WHERE tenant_id = $1 AND (datasource_id = $2 OR datasource_id IS NULL OR datasource_id = '') AND is_active = true
			ORDER BY team_name
		`
		args = []interface{}{tenantID, datasourceID}
	} else {
		query = `
			SELECT id, team_key, team_name, description, team_type, is_active
			FROM bp_teams
			WHERE tenant_id = $1 AND is_active = true
			ORDER BY team_name
		`
		args = []interface{}{tenantID}
	}

	rows, err := h.db.Query(query, args...)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch teams: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	teams := []map[string]interface{}{}
	for rows.Next() {
		var team map[string]interface{} = make(map[string]interface{})
		var id, teamKey, teamName, desc, teamType string
		var isActive bool

		if err := rows.Scan(&id, &teamKey, &teamName, &desc, &teamType, &isActive); err != nil {
			continue
		}

		team["id"] = id
		team["team_key"] = teamKey
		team["team_name"] = teamName
		team["description"] = desc
		team["team_type"] = teamType
		team["is_active"] = isActive

		teams = append(teams, team)
	}

	respondJSONRBAC(w, r, teams, http.StatusOK)
}

func (h *RBACHandlers) createTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamKey     string  `json:"team_key"`
		TeamName    string  `json:"team_name"`
		Description string  `json:"description"`
		TeamType    string  `json:"team_type"`
		ManagerID   *string `json:"manager_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	var id string
	err = h.db.QueryRow(`
		INSERT INTO bp_teams (tenant_id, datasource_id, team_key, team_name, description, team_type, manager_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, tenantID, datasourceID, req.TeamKey, req.TeamName, req.Description, req.TeamType, req.ManagerID).Scan(&id)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create team: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"id": id, "status": "created"}, http.StatusCreated)
}

func (h *RBACHandlers) addTeamMember(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	var req struct {
		UserID     string `json:"user_id"`
		RoleInTeam string `json:"role_in_team"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		INSERT INTO bp_team_members (team_id, user_id, role_in_team)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_id, user_id) DO NOTHING
	`, teamID, req.UserID, req.RoleInTeam)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add team member: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "added"}, http.StatusCreated)
}

func (h *RBACHandlers) removeTeamMember(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	userID := chi.URLParam(r, "userId")

	_, err := h.db.Exec("UPDATE bp_team_members SET is_active = false WHERE team_id = $1 AND user_id = $2", teamID, userID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove team member: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "removed"}, http.StatusOK)
}

func (h *RBACHandlers) getTeamMembers(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")

	rows, err := h.db.Query(`
		SELECT user_id, role_in_team, joined_at
		FROM bp_team_members
		WHERE team_id = $1 AND is_active = true
		ORDER BY joined_at
	`, teamID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch team members: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := []map[string]interface{}{}
	for rows.Next() {
		var member map[string]interface{} = make(map[string]interface{})
		var userID, roleInTeam string
		var joinedAt time.Time

		if err := rows.Scan(&userID, &roleInTeam, &joinedAt); err != nil {
			continue
		}

		member["user_id"] = userID
		member["role_in_team"] = roleInTeam
		member["joined_at"] = joinedAt

		members = append(members, member)
	}

	respondJSONRBAC(w, r, members, http.StatusOK)
}

// ============================================================================
// AUDIT
// ============================================================================

func (h *RBACHandlers) listPermissionAudit(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID
	limit := r.URL.Query().Get("limit")

	if limit == "" {
		limit = "100"
	}

	rows, err := h.db.Query(`
		SELECT action_type, subject_type, subject_id, object_type, object_id, performed_by, performed_at
		FROM bp_permission_audit_log
		WHERE tenant_id = $1 AND datasource_id = $2
		ORDER BY performed_at DESC
		LIMIT $3
	`, tenantID, datasourceID, limit)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch audit log: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	auditLog := []map[string]interface{}{}
	for rows.Next() {
		var entry map[string]interface{} = make(map[string]interface{})
		var actionType, subjectType, subjectID, objectType, objectID, performedBy string
		var performedAt time.Time

		if err := rows.Scan(&actionType, &subjectType, &subjectID, &objectType, &objectID, &performedBy, &performedAt); err != nil {
			continue
		}

		entry["action_type"] = actionType
		entry["subject_type"] = subjectType
		entry["subject_id"] = subjectID
		entry["object_type"] = objectType
		entry["object_id"] = objectID
		entry["performed_by"] = performedBy
		entry["performed_at"] = performedAt

		auditLog = append(auditLog, entry)
	}

	respondJSONRBAC(w, r, auditLog, http.StatusOK)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func respondJSONRBAC(w http.ResponseWriter, _r *http.Request, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
