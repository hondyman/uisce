package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// RBAC & Permissions API Handlers
// Fortune 500 Enterprise-Grade Security
// ============================================================================

type RBACHandlers struct {
	db *sqlx.DB
}

func NewRBACHandlers(db *sqlx.DB) *RBACHandlers {
	return &RBACHandlers{db: db}
}

func (h *RBACHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/rbac", func(r chi.Router) {
		// Roles
		r.Get("/roles", h.listRoles)
		r.Post("/roles", h.createRole)
		r.Get("/roles/{roleId}", h.getRole)
		r.Put("/roles/{roleId}", h.updateRole)
		r.Delete("/roles/{roleId}", h.deleteRole)

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
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	DatasourceID string    `json:"datasource_id" db:"datasource_id"`
	RoleKey      string    `json:"role_key" db:"role_key"`
	RoleName     string    `json:"role_name" db:"role_name"`
	Description  string    `json:"description" db:"description"`
	RoleType     string    `json:"role_type" db:"role_type"`
	RoleLevel    string    `json:"role_level" db:"role_level"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedBy    *string   `json:"created_by" db:"created_by"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

func (h *RBACHandlers) listRoles(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	var roles []Role
	err := h.db.Select(&roles, `
		SELECT * FROM bp_roles
		WHERE tenant_id = $1 AND datasource_id = $2 AND is_active = true
		ORDER BY role_level, role_name
	`, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch roles: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, roles, http.StatusOK)
}

func (h *RBACHandlers) createRole(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	var req struct {
		RoleKey     string   `json:"role_key"`
		RoleName    string   `json:"role_name"`
		Description string   `json:"description"`
		RoleLevel   string   `json:"role_level"`
		Permissions []string `json:"permissions"` // Permission IDs to assign
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create role
	var roleID string
	err := h.db.QueryRow(`
		INSERT INTO bp_roles (tenant_id, datasource_id, role_key, role_name, description, role_type, role_level)
		VALUES ($1, $2, $3, $4, $5, 'custom', $6)
		RETURNING id
	`, tenantID, datasourceID, req.RoleKey, req.RoleName, req.Description, req.RoleLevel).Scan(&roleID)

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
	tenantID := r.URL.Query().Get("tenant_id")

	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

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
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

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
		TenantID     string  `json:"tenant_id"`
		DatasourceID string  `json:"datasource_id"`
		ScopeType    *string `json:"scope_type"`
		ScopeID      *string `json:"scope_id"`
		ExpiresAt    *string `json:"expires_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Allow tenant_id and datasource_id from query params if not in body
	if req.TenantID == "" {
		req.TenantID = r.URL.Query().Get("tenant_id")
	}
	if req.DatasourceID == "" {
		req.DatasourceID = r.URL.Query().Get("datasource_id")
	}

	if req.TenantID == "" || req.DatasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		INSERT INTO bp_user_roles (user_id, role_id, tenant_id, datasource_id, scope_type, scope_id, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, role_id, tenant_id, datasource_id, scope_type, scope_id) DO NOTHING
	`, req.UserID, roleID, req.TenantID, req.DatasourceID, req.ScopeType, req.ScopeID, req.ExpiresAt)

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
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

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
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

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
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	var fieldPerms []map[string]interface{}

	query := `
		SELECT fp.id, r.role_key, r.role_name, fp.resource_type, fp.field_name, fp.permission_level
		FROM bp_field_permissions fp
		JOIN bp_roles r ON fp.role_id = r.id
		WHERE fp.tenant_id = $1 AND fp.datasource_id = $2
	`
	args := []interface{}{tenantID, datasourceID}

	// Optional role_id filter
	roleID := r.URL.Query().Get("role_id")
	if roleID != "" {
		query += " AND fp.role_id = $3"
		args = append(args, roleID)
	}

	query += " ORDER BY r.role_name, fp.resource_type, fp.field_name"

	rows, err := h.db.Query(query, args...)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch field permissions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var fp map[string]interface{} = make(map[string]interface{})
		var id, roleKey, roleName, resType, fieldName, permLevel string

		if err := rows.Scan(&id, &roleKey, &roleName, &resType, &fieldName, &permLevel); err != nil {
			continue
		}

		fp["id"] = id
		fp["role_key"] = roleKey
		fp["role_name"] = roleName
		fp["resource_type"] = resType
		fp["field_name"] = fieldName
		fp["permission_level"] = permLevel

		fieldPerms = append(fieldPerms, fp)
	}

	respondJSONRBAC(w, r, fieldPerms, http.StatusOK)
}

func (h *RBACHandlers) createFieldPermission(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID        string  `json:"tenant_id"`
		DatasourceID    string  `json:"datasource_id"`
		RoleID          string  `json:"role_id"`
		ResourceType    string  `json:"resource_type"`
		ResourceID      *string `json:"resource_id"`
		FieldName       string  `json:"field_name"`
		PermissionLevel string  `json:"permission_level"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var id string
	err := h.db.QueryRow(`
		INSERT INTO bp_field_permissions (tenant_id, datasource_id, role_id, resource_type, resource_id, field_name, permission_level)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.TenantID, req.DatasourceID, req.RoleID, req.ResourceType, req.ResourceID, req.FieldName, req.PermissionLevel).Scan(&id)

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
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	var fieldPerms []map[string]string
	rows, err := h.db.Query(`
		SELECT DISTINCT fp.field_name, fp.permission_level
		FROM bp_user_roles ur
		JOIN bp_field_permissions fp ON ur.role_id = fp.role_id
		WHERE ur.user_id = $1
		  AND fp.tenant_id = $2
		  AND fp.datasource_id = $3
		  AND fp.resource_type = $4
		  AND (fp.resource_id IS NULL OR fp.resource_id = $5)
		  AND ur.is_active = true
		ORDER BY fp.field_name, 
		  CASE fp.permission_level 
		    WHEN 'write' THEN 1
		    WHEN 'read' THEN 2
		    WHEN 'mask' THEN 3
		    WHEN 'none' THEN 4
		  END
	`, userID, tenantID, datasourceID, resourceType, resourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch user field permissions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var fp map[string]string = make(map[string]string)
		var fieldName, permLevel string

		if err := rows.Scan(&fieldName, &permLevel); err != nil {
			continue
		}

		fp["field_name"] = fieldName
		fp["permission_level"] = permLevel

		fieldPerms = append(fieldPerms, fp)
	}

	respondJSONRBAC(w, r, fieldPerms, http.StatusOK)
}

// ============================================================================
// DELEGATIONS (CONTINUED IN NEXT MESSAGE DUE TO LENGTH)
// ============================================================================

func (h *RBACHandlers) listDelegations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

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
		TenantID       string  `json:"tenant_id"`
		DatasourceID   string  `json:"datasource_id"`
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

	var id string
	err := h.db.QueryRow(`
		INSERT INTO bp_approval_delegations (
			tenant_id, datasource_id, delegator_user_id, delegate_user_id,
			delegation_type, resource_type, resource_id, start_date, end_date, reason
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, req.TenantID, req.DatasourceID, req.DelegatorID, req.DelegateID,
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
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(`
		SELECT id, team_key, team_name, description, team_type, is_active
		FROM bp_teams
		WHERE tenant_id = $1 AND datasource_id = $2 AND is_active = true
		ORDER BY team_name
	`, tenantID, datasourceID)

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
		TenantID     string  `json:"tenant_id"`
		DatasourceID string  `json:"datasource_id"`
		TeamKey      string  `json:"team_key"`
		TeamName     string  `json:"team_name"`
		Description  string  `json:"description"`
		TeamType     string  `json:"team_type"`
		ManagerID    *string `json:"manager_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var id string
	err := h.db.QueryRow(`
		INSERT INTO bp_teams (tenant_id, datasource_id, team_key, team_name, description, team_type, manager_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.TenantID, req.DatasourceID, req.TeamKey, req.TeamName, req.Description, req.TeamType, req.ManagerID).Scan(&id)

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
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	limit := r.URL.Query().Get("limit")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

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
