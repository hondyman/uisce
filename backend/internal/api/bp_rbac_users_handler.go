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

	if tenantID != "" {
		query = `
			SELECT id, username, email, name, first_name, last_name, status, is_active, created_at, tenant_id
			FROM users
			WHERE tenant_id = $1 OR tenant_id IS NULL
			ORDER BY name, username
		`
		args = []interface{}{tenantID}
	} else {
		query = `
			SELECT id, username, email, name, first_name, last_name, status, is_active, created_at, tenant_id
			FROM users
			ORDER BY name, username
		`
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

	var userID string
	err := h.db.QueryRow(`
		INSERT INTO users (username, email, name, first_name, last_name, tenant_id, status, is_active, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8)
		RETURNING id
	`, req.Username, req.Email, req.Name, req.FirstName, req.LastName, req.TenantID, req.Status, req.Password).Scan(&userID)

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

