package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// listUsers returns all users for role assignment
func (h *RBACHandlers) listUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	var users []map[string]interface{}
	var query string
	var args []interface{}

	// Query only columns that exist in the actual public.users schema.
	baseQuery := `
		SELECT id, username, email, name, role, organization,
		       is_core_admin, is_active, tenant_id, created_at, last_login
		FROM users
	`
	if tenantID != "" {
		query = baseQuery + ` WHERE tenant_id = $1 OR tenant_id IS NULL ORDER BY username`
		args = []interface{}{tenantID}
	} else {
		query = baseQuery + ` ORDER BY username`
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch users: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user map[string]interface{} = make(map[string]interface{})
		var id, username, email string
		var name, role, organization, userTenantID sql.NullString
		var isCoreAdmin, isActive bool
		var createdAt time.Time
		var lastLogin sql.NullTime

		if err := rows.Scan(&id, &username, &email, &name, &role, &organization, &isCoreAdmin, &isActive, &userTenantID, &createdAt, &lastLogin); err != nil {
			fmt.Printf("[WARN] listUsers scan error: %v\n", err)
			continue
		}

		user["id"] = id
		user["username"] = username
		user["email"] = email
		if name.Valid && name.String != "" {
			user["name"] = name.String
		} else if email != "" {
			user["name"] = email
		} else {
			user["name"] = username
		}
		if role.Valid && role.String != "" {
			user["role"] = role.String
		}
		if organization.Valid && organization.String != "" {
			user["organization"] = organization.String
		}
		user["is_core_admin"] = isCoreAdmin
		user["is_active"] = isActive
		user["created_at"] = createdAt
		if lastLogin.Valid {
			user["last_login"] = lastLogin.Time
		}
		if userTenantID.Valid {
			user["tenant_id"] = userTenantID.String
		} else {
			user["tenant_id"] = nil
		}

		users = append(users, user)
	}

	respondJSONRBAC(w, r, users, http.StatusOK)
}

// createUser creates a new user in the tenant
func (h *RBACHandlers) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username  string `json:"username"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		TenantID  string `json:"tenant_id"`
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

	// Use query param tenant_id if not in body
	if req.TenantID == "" {
		req.TenantID = r.URL.Query().Get("tenant_id")
	}

	if req.TenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	// Default status
	if req.Status == "" {
		req.Status = "active"
	}

	// public.users is a view over app_user; writes must go to the underlying table.
	var userID string
	err := h.db.QueryRow(`
		INSERT INTO app_user (id, username, email, name, display_name, tenant_id, is_active, password_hash, created_at)
		VALUES (gen_random_uuid()::text, $1, $2, $3, $4, $5, true, $6, CURRENT_TIMESTAMP)
		RETURNING id
	`, req.Username, req.Email, req.Name, req.Name, req.TenantID, req.Password).Scan(&userID)

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

	var tenantVal any
	if req.TenantID == nil || *req.TenantID == "" {
		tenantVal = nil
	} else {
		tenantVal = *req.TenantID
	}

	_, err := h.db.Exec(`
		UPDATE app_user
		SET tenant_id = $1, updated_at_time = now()
		WHERE id = $2
	`, tenantVal, userID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user tenant assignment: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSONRBAC(w, r, map[string]string{"status": "updated"}, http.StatusOK)
}

