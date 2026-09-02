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
// searchUsers backs the role-assignment "search for a user" picker.
func (h *RBACHandlers) searchUsers(w http.ResponseWriter, r *http.Request) {
	if _, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps); err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		respondJSONRBAC(w, r, []map[string]interface{}{}, http.StatusOK)
		return
	}

	rows, err := h.db.Query(`
		SELECT id, username, email, name
		FROM users
		WHERE is_active = true
		  AND (email ILIKE $1 OR username ILIKE $1 OR name ILIKE $1)
		ORDER BY name, username
		LIMIT 20
	`, "%"+q+"%")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, email string
		var username, name sql.NullString
		if err := rows.Scan(&id, &username, &email, &name); err != nil {
			continue
		}
		user := map[string]interface{}{"id": id, "email": email}
		if username.Valid {
			user["username"] = username.String
		} else {
			user["username"] = email
		}
		if name.Valid {
			user["name"] = name.String
		}
		users = append(users, user)
	}
	if users == nil {
		users = []map[string]interface{}{}
	}

	respondJSONRBAC(w, r, users, http.StatusOK)
}

func (h *RBACHandlers) listUsers(w http.ResponseWriter, r *http.Request) {
	_, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Tenant scoping for user-to-tenant access is handled via the user_tenant
	// mapping table, not app_user.tenant_id (a legacy single-tenant field).
	// This endpoint feeds the tenant assignment picker, so it must list every
	// active user regardless of their home tenant_id.
	var users []map[string]interface{}
	query := `
		SELECT id, username, email, name, first_name, last_name, status, is_active, created_at, tenant_id
		FROM users
		WHERE is_active = true
		ORDER BY name, username
	`

	rows, err := h.db.Query(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user map[string]interface{} = make(map[string]interface{})
		var id, email string
		// username is nullable at the schema level (e.g. a user provisioned
		// from a JWT with no natural username claim) — scanning it as a
		// plain string previously made rows.Scan fail and silently drop the
		// row from this admin picker, hiding the user from role assignment.
		var username, name, firstName, lastName, status, userTenantID sql.NullString
		var isActive bool
		var createdAt time.Time

		if err := rows.Scan(&id, &username, &email, &name, &firstName, &lastName, &status, &isActive, &createdAt, &userTenantID); err != nil {
			continue
		}

		user["id"] = id
		if username.Valid {
			user["username"] = username.String
		} else {
			user["username"] = email
		}
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
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
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
	}

	var tenantVal any
	if req.TenantID == nil || *req.TenantID == "" {
		tenantVal = nil
	} else {
		tenantVal = *req.TenantID
	}

	_, err = h.db.Exec(`
		UPDATE users
		SET tenant_id = $1, updated_at = now()
		WHERE id = $2
	`, tenantVal, userID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user tenant assignment: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "updated"}, http.StatusOK)
}

