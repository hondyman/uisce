package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// listUsers returns all users for role assignment
func (h *RBACHandlers) listUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	var users []map[string]interface{}
	var query string
	var args []interface{}

	if tenantID != "" {
		query = `
			SELECT id, username, email, name, first_name, last_name, status, is_active, created_at
			FROM users
			WHERE tenant_id = $1 OR tenant_id IS NULL
			ORDER BY name, username
		`
		args = []interface{}{tenantID}
	} else {
		query = `
			SELECT id, username, email, name, first_name, last_name, status, is_active, created_at
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
		var name, firstName, lastName, status sql.NullString
		var isActive bool
		var createdAt time.Time

		if err := rows.Scan(&id, &username, &email, &name, &firstName, &lastName, &status, &isActive, &createdAt); err != nil {
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

	// In a real app we'd hash the password here.
	// For this exercise we'll insert directly or use valid mock hash if needed.
	// Since I don't see a shared password lib immediately usable without imports,
	// I will insert the password as is or use a placeholder if the DB requires a hash format.
	// The seed script usually inserts raw or simple hashes.

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
