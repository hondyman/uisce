package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

// isTenantOrGlobalAdmin reports whether the caller holds a role authorized to
// manage RBAC/tenant assignments. Mirrors handlers.hasAdminRole.
func isTenantOrGlobalAdmin(roles []string) bool {
	for _, role := range roles {
		switch strings.ToUpper(strings.TrimSpace(role)) {
		case "GLOBAL_OPS", "TENANT_ADMIN", "ADMIN":
			return true
		}
	}
	return false
}

// listUsers returns all users for role assignment
// requireTenantAdmin answers 403 unless the caller administers the tenant they act for.
// Ordinary members do not see the users of their tenant, and nobody sees other tenants'.
func requireTenantAdmin(w http.ResponseWriter, secCtx *security.Context) bool {
	if secCtx.IsGlobalAdmin || isTenantOrGlobalAdmin(secCtx.Roles) {
		return true
	}
	http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
	return false
}

// listUsers returns the users of the caller's tenant: the users whose home tenant is
// it, or who are mapped to it. This is the admin screens' list, so it carries the
// fields those screens render, email included, for members only. Users of other
// tenants are never returned, and users with no home tenant are not here: the
// assignment picker (listAssignableUsers) is where they appear.
func (h *RBACHandlers) listUsers(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		handlers.WriteSecurityError(w, err)
		return
	}
	if !requireTenantAdmin(w, secCtx) {
		return
	}

	var users []map[string]interface{}
	rows, err := h.db.Query(`
		SELECT u.id, u.username, u.email, u.name, u.first_name, u.last_name, u.status, u.is_active, u.created_at, u.tenant_id
		FROM users u
		WHERE u.is_active = true
		  AND (u.tenant_id = $1::uuid
		       OR EXISTS (SELECT 1 FROM user_tenant ut WHERE ut.user_id = u.id AND ut.tenant_id = $1::uuid))
		ORDER BY u.name, u.username
	`, secCtx.TenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user map[string]interface{} = make(map[string]interface{})
		var id, username, email string
		var name, firstName, lastName, status, userTenantID sql.NullString
		var isActive bool
		var createdAt time.Time

		if err := rows.Scan(&id, &username, &email, &name, &firstName, &lastName, &status, &isActive, &createdAt, &userTenantID); err != nil {
			continue
		}

		user["id"] = id
		user["username"] = username
		user["email"] = email
		if name.Valid {
			user["name"] = name.String
		}
		if firstName.Valid {
			user["first_name"] = firstName.String
		}
		if lastName.Valid {
			user["last_name"] = lastName.String
		}
		if status.Valid {
			user["status"] = status.String
		} else {
			user["status"] = "active"
		}
		user["is_active"] = isActive
		user["created_at"] = createdAt
		if userTenantID.Valid {
			user["tenant_id"] = userTenantID.String
		} else {
			user["tenant_id"] = nil
		}

		users = append(users, user)
	}

	respondJSONRBAC(w, r, users, http.StatusOK)
}

// listAssignableUsers is the tenant-assignment picker. It offers the users who can be
// given access to the caller's tenant: its members, and users with no home tenant and
// no tenant mapping at all (the same rule updateUserTenant enforces). Email is sent for
// members only; an unassigned user is identified by username. Whether unassigned users
// should carry email is a product decision, so this is the one place to change it.
func (h *RBACHandlers) listAssignableUsers(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		handlers.WriteSecurityError(w, err)
		return
	}
	if !requireTenantAdmin(w, secCtx) {
		return
	}

	rows, err := h.db.Query(`
		SELECT u.id, u.username, u.email, u.name,
		       (u.tenant_id = $1::uuid OR EXISTS (SELECT 1 FROM user_tenant ut WHERE ut.user_id = u.id AND ut.tenant_id = $1::uuid)) AS is_member
		FROM users u
		WHERE u.is_active = true
		  AND (u.tenant_id = $1::uuid
		       OR EXISTS (SELECT 1 FROM user_tenant ut WHERE ut.user_id = u.id AND ut.tenant_id = $1::uuid)
		       OR (u.tenant_id IS NULL AND NOT EXISTS (SELECT 1 FROM user_tenant ut2 WHERE ut2.user_id = u.id)))
		ORDER BY u.name, u.username
	`, secCtx.TenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch assignable users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []map[string]interface{}{}
	for rows.Next() {
		var id, username string
		var email, name sql.NullString
		var isMember bool
		if err := rows.Scan(&id, &username, &email, &name, &isMember); err != nil {
			continue
		}
		user := map[string]interface{}{"id": id, "username": username, "is_member": isMember}
		if name.Valid {
			user["name"] = name.String
		}
		if isMember && email.Valid {
			user["email"] = email.String
		}
		users = append(users, user)
	}
	respondJSONRBAC(w, r, users, http.StatusOK)
}

// createUser creates a new user in the tenant
func (h *RBACHandlers) createUser(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		handlers.WriteSecurityError(w, err)
		return
	}

	var req struct {
		Username  string `json:"username"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
		Status    string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" {
		http.Error(w, "Username and Email are required", http.StatusBadRequest)
		return
	}

	tenantID := secCtx.TenantID

	// Default status
	if req.Status == "" {
		req.Status = "active"
	}

	var userID string
	err = h.db.QueryRow(`
		INSERT INTO users (username, email, name, first_name, last_name, tenant_id, status, is_active, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8)
		RETURNING id
	`, req.Username, req.Email, req.Name, req.FirstName, req.LastName, tenantID, req.Status, req.Password).Scan(&userID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"id": userID, "status": "created"}, http.StatusCreated)
}

// updateUserTenant updates the tenant_id for a user
func (h *RBACHandlers) updateUserTenant(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	
	var req struct {
		TenantID *string `json:"tenant_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		handlers.WriteSecurityError(w, err)
		return
	}

	// Only global admins may move a user to an arbitrary tenant. Tenant-scoped
	// admins may only assign users into their own tenant (or clear the
	// assignment), never into a tenant they don't administer.
	if !secCtx.IsGlobalAdmin {
		if !isTenantOrGlobalAdmin(secCtx.Roles) {
			http.Error(w, "Forbidden: admin role required", http.StatusForbidden)
			return
		}
		if req.TenantID != nil && *req.TenantID != "" && *req.TenantID != secCtx.TenantID {
			http.Error(w, "Forbidden: cannot assign user to a different tenant", http.StatusForbidden)
			return
		}
		// A tenant admin acts on the users of their own tenant, and may bring in a user who
		// has no home tenant yet. Any other user is not found for them, so the tenant
		// assignments of other tenants cannot be read or changed through this route.
		if err := authorizeUser(r.Context(), h.db, secCtx.TenantID, userID); err != nil {
			var unassigned bool
			if qErr := h.db.GetContext(r.Context(), &unassigned,
				`SELECT EXISTS (SELECT 1 FROM app_user u WHERE u.id = $1 AND u.tenant_id IS NULL
				                AND NOT EXISTS (SELECT 1 FROM user_tenant ut WHERE ut.user_id = u.id))`, userID); qErr != nil {
				http.Error(w, fmt.Sprintf("Failed to check user: %v", qErr), http.StatusInternalServerError)
				return
			}
			if !unassigned {
				writeScopeError(w, err)
				return
			}
		}
	}

	var tenantVal any
	if req.TenantID == nil || *req.TenantID == "" {
		tenantVal = nil
	} else {
		tenantVal = *req.TenantID
	}

	// Non-global admins write only where the user is unassigned or already theirs.
	res, err := h.db.Exec(`
		UPDATE users
		SET tenant_id = $1, updated_at = now()
		WHERE id = $2 AND ($3 OR tenant_id IS NULL OR tenant_id = $4)
	`, tenantVal, userID, secCtx.IsGlobalAdmin, secCtx.TenantID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user tenant assignment: %v", err), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeScopeError(w, errNotInTenant)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "updated"}, http.StatusOK)
}

