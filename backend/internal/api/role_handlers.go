package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/logging"
	appmid "github.com/hondyman/semlayer/backend/internal/middleware"
)

// Role Management API Handlers

// CreateRoleRequest represents a request to create a new role
type CreateRoleRequest struct {
	RoleName      string   `json:"role_name"`
	Description   string   `json:"description"`
	IsGlobalAdmin bool     `json:"is_global_admin"`
	Permissions   []string `json:"permissions"`
}

// AssignRoleRequest represents a request to assign a role to a user
type AssignRoleRequest struct {
	RoleID string `json:"role_id"`
}

// createRole creates a new role in the IAM system
func (s *Server) createRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current user from context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := user.ID
	tenantID := user.TenantID

	// Create role in IAM schema
	var roleID string
	err := s.DB.QueryRowContext(r.Context(), `
		INSERT INTO iam.roles (tenant_id, role_name, description, is_global_admin, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING role_id
	`, tenantID, req.RoleName, req.Description, req.IsGlobalAdmin, userID).Scan(&roleID)

	if err != nil {
		http.Error(w, "Failed to create role", http.StatusInternalServerError)
		return
	}

	// Create security event (will be picked up by Debezium)
	eventPayload := map[string]interface{}{
		"role_id":         roleID,
		"role_name":       req.RoleName,
		"tenant_id":       tenantID,
		"is_global_admin": req.IsGlobalAdmin,
	}
	payloadJSON, _ := json.Marshal(eventPayload)

	_, err = s.DB.ExecContext(r.Context(), `
		SELECT iam.create_security_event('role_created', 'role', $1, $2, $3, $4)
	`, roleID, tenantID, payloadJSON, userID)

	if err != nil {
		// Log error but don't fail the request
		logging.GetLogger().Sugar().Errorf("Failed to create security event: %v", err)
	}

	respond(w, r, map[string]string{
		"role_id": roleID,
		"status":  "created",
	}, nil)
}

// listRoles lists all roles for the current tenant
func (s *Server) listRoles(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tenantID := user.TenantID
	isGlobalAdmin := user.IsCoreAdmin // Assuming IsCoreAdmin maps to global admin or check role
	// If IsCoreAdmin is not what we want, check user.Role == "admin" && user.Organization == "uisce"
	if user.Organization == "uisce" && user.Role == "admin" {
		isGlobalAdmin = true
	}

	// Query roles
	var rows *sql.Rows
	var err error
	if isGlobalAdmin {
		// Global admins see all roles
		rows, err = s.DB.QueryContext(r.Context(), `
			SELECT role_id, tenant_id, role_name, description, is_global_admin, created_at
			FROM iam.roles
			ORDER BY created_at DESC
		`)
	} else {
		// Regular users only see their tenant's roles
		rows, err = s.DB.QueryContext(r.Context(), `
			SELECT role_id, tenant_id, role_name, description, is_global_admin, created_at
			FROM iam.roles
			WHERE tenant_id = $1
			ORDER BY created_at DESC
		`, tenantID)
	}

	if err != nil {
		http.Error(w, "Failed to query roles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var roles []map[string]interface{}
	for rows.Next() {
		var role struct {
			RoleID        string
			TenantID      string
			RoleName      string
			Description   sql.NullString
			IsGlobalAdmin bool
			CreatedAt     string
		}

		err := rows.Scan(&role.RoleID, &role.TenantID, &role.RoleName, &role.Description, &role.IsGlobalAdmin, &role.CreatedAt)
		if err != nil {
			continue
		}

		roleMap := map[string]interface{}{
			"role_id":         role.RoleID,
			"tenant_id":       role.TenantID,
			"role_name":       role.RoleName,
			"is_global_admin": role.IsGlobalAdmin,
			"created_at":      role.CreatedAt,
		}

		if role.Description.Valid {
			roleMap["description"] = role.Description.String
		}

		roles = append(roles, roleMap)
	}

	respond(w, r, roles, nil)
}

// assignRole assigns a role to a user
func (s *Server) assignRole(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "user_id")

	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current user ID
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID

	// Assign role
	_, err := s.DB.ExecContext(r.Context(), `
		INSERT INTO iam.user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, targetUserID, req.RoleID, currentUserID)

	if err != nil {
		http.Error(w, "Failed to assign role", http.StatusInternalServerError)
		return
	}

	// Create security event
	eventPayload := map[string]interface{}{
		"user_id": targetUserID,
		"role_id": req.RoleID,
	}
	payloadJSON, _ := json.Marshal(eventPayload)

	_, err = s.DB.ExecContext(r.Context(), `
		SELECT iam.create_security_event('user_role_assigned', 'user_role', $1, NULL, $2, $3)
	`, targetUserID, payloadJSON, currentUserID)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create security event: %v", err)
	}

	respond(w, r, map[string]string{
		"status": "assigned",
	}, nil)
}

// revokeRole revokes a role from a user
func (s *Server) revokeRole(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "user_id")
	roleID := chi.URLParam(r, "role_id")

	// Get current user ID
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	currentUserID := user.ID

	// Revoke role
	_, err := s.DB.ExecContext(r.Context(), `
		DELETE FROM iam.user_roles
		WHERE user_id = $1 AND role_id = $2
	`, targetUserID, roleID)

	if err != nil {
		http.Error(w, "Failed to revoke role", http.StatusInternalServerError)
		return
	}

	// Create security event
	eventPayload := map[string]interface{}{
		"user_id": targetUserID,
		"role_id": roleID,
	}
	payloadJSON, _ := json.Marshal(eventPayload)

	_, err = s.DB.ExecContext(r.Context(), `
		SELECT iam.create_security_event('user_role_revoked', 'user_role', $1, NULL, $2, $3)
	`, targetUserID, payloadJSON, currentUserID)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to create security event: %v", err)
	}

	respond(w, r, map[string]string{
		"status": "revoked",
	}, nil)
}

// getUserRoles gets all roles for a user
func (s *Server) getUserRoles(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "user_id")

	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT r.role_id, r.role_name, r.description, r.is_global_admin, ur.assigned_at
		FROM iam.user_roles ur
		JOIN iam.roles r ON ur.role_id = r.role_id
		WHERE ur.user_id = $1
		ORDER BY ur.assigned_at DESC
	`, targetUserID)

	if err != nil {
		http.Error(w, "Failed to query user roles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var roles []map[string]interface{}
	for rows.Next() {
		var role struct {
			RoleID        string
			RoleName      string
			Description   sql.NullString
			IsGlobalAdmin bool
			AssignedAt    string
		}

		err := rows.Scan(&role.RoleID, &role.RoleName, &role.Description, &role.IsGlobalAdmin, &role.AssignedAt)
		if err != nil {
			continue
		}

		roleMap := map[string]interface{}{
			"role_id":         role.RoleID,
			"role_name":       role.RoleName,
			"is_global_admin": role.IsGlobalAdmin,
			"assigned_at":     role.AssignedAt,
		}

		if role.Description.Valid {
			roleMap["description"] = role.Description.String
		}

		roles = append(roles, roleMap)
	}

	respond(w, r, roles, nil)
}

// RegisterRoleRoutes registers role management routes
func (s *Server) RegisterRoleRoutes(r chi.Router) {
	r.Route("/roles", func(r chi.Router) {
		r.Use(appmid.SessionAuthMiddleware(appmid.SessionAuthConfig{DB: s.DB, SessionCookie: "session_token", AllowBearerFallback: true})) // Require authentication

		r.Post("/", s.createRole)
		r.Get("/", s.listRoles)
	})

	r.Route("/users/{user_id}/roles", func(r chi.Router) {
		r.Use(appmid.SessionAuthMiddleware(appmid.SessionAuthConfig{DB: s.DB, SessionCookie: "session_token", AllowBearerFallback: true}))

		r.Get("/", s.getUserRoles)
		r.Post("/", s.assignRole)
		r.Delete("/{role_id}", s.revokeRole)
	})
}

// listIAMEvents returns a list of security audit events
func (s *Server) listIAMEvents(w http.ResponseWriter, r *http.Request) {
	// Auth check (admins only)
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Allow global admins and tenant admins
	// If tenant admin, filter by tenant_id? Yes.
	// iam.security_events has tenant_id.

	var rows *sql.Rows
	var err error

	limit := 100 // Hardcoded limit for now, or get from query param

	if user.IsCoreAdmin || (user.Organization == "uisce" && user.Role == "admin") {
		// Global admin sees all? Or maybe scoped if requested.
		// Let's show all for global admin for now.
		rows, err = s.DB.QueryContext(r.Context(), `
            SELECT event_id, event_type, entity_type, entity_id::text, actor_id, payload, created_at, tenant_id
            FROM iam.security_events
            ORDER BY created_at DESC
            LIMIT $1
        `, limit)
	} else {
		// Tenant admin
		rows, err = s.DB.QueryContext(r.Context(), `
            SELECT event_id, event_type, entity_type, entity_id::text, actor_id, payload, created_at, tenant_id
            FROM iam.security_events
            WHERE tenant_id = $1
            ORDER BY created_at DESC
            LIMIT $2
        `, user.TenantID, limit)
	}

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to query security events: %v", err)
		http.Error(w, "Failed to query security events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var evt struct {
			EventID    string
			EventType  string
			EntityType string
			EntityID   string
			ActorID    string
			Payload    []byte
			CreatedAt  time.Time
			TenantID   string
		}
		if err := rows.Scan(&evt.EventID, &evt.EventType, &evt.EntityType, &evt.EntityID, &evt.ActorID, &evt.Payload, &evt.CreatedAt, &evt.TenantID); err != nil {
			continue
		}

		var payloadMap map[string]interface{}
		_ = json.Unmarshal(evt.Payload, &payloadMap)

		events = append(events, map[string]interface{}{
			"event_id":    evt.EventID,
			"event_type":  evt.EventType,
			"entity_type": evt.EntityType,
			"entity_id":   evt.EntityID,
			"actor_id":    evt.ActorID,
			"payload":     payloadMap,
			"created_at":  evt.CreatedAt,
			"tenant_id":   evt.TenantID,
		})
	}

	respond(w, r, events, nil)
}

// getSecurityStats returns summary statistics for the security dashboard
func (s *Server) getSecurityStats(w http.ResponseWriter, r *http.Request) {
	// Auth check
	user, ok := auth.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	var stats struct {
		TotalUsers     int    `json:"total_users"`
		ActiveSessions int    `json:"active_sessions"`
		ActiveRoles    int    `json:"active_roles"`
		RecentAlerts   int    `json:"recent_alerts"`
		SyncStatus     string `json:"sync_status"`
		LastSyncTime   string `json:"last_sync_time"`
	}

	// 1. Total Users
	if user.IsCoreAdmin || (user.Organization == "uisce" && user.Role == "admin") {
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM public.users WHERE is_active = true").Scan(&stats.TotalUsers); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to count users: %v", err)
		}
	} else {
		// Filter by organization/tenant. Assuming organization matches tenant boundary for now or add tenant_id check if available.
		// Since we have user.TenantID, let's use it if we can join or if users has it.
		// For safety, let's check Organization.
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM public.users WHERE is_active = true AND organization = $1", user.Organization).Scan(&stats.TotalUsers); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to count users: %v", err)
		}
	}

	// 2. Active Sessions (active in last 24 hours)
	// For now return 0 for non-admins as session table is simple
	if user.IsCoreAdmin {
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM private_markets_sessions WHERE generated_at > NOW() - INTERVAL '24 hours'").Scan(&stats.ActiveSessions); err != nil {
			stats.ActiveSessions = 0
		}
	}

	// 3. Active Roles
	if user.IsCoreAdmin || (user.Organization == "uisce" && user.Role == "admin") {
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM iam.roles").Scan(&stats.ActiveRoles); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to count roles: %v", err)
		}
	} else {
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM iam.roles WHERE tenant_id = $1", user.TenantID).Scan(&stats.ActiveRoles); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to count roles: %v", err)
		}
	}

	// 4. Recent Alerts (last 24 hours)
	if user.IsCoreAdmin || (user.Organization == "uisce" && user.Role == "admin") {
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM iam.security_events WHERE created_at > NOW() - INTERVAL '24 hours'").Scan(&stats.RecentAlerts); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to count alerts: %v", err)
		}
	} else {
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM iam.security_events WHERE created_at > NOW() - INTERVAL '24 hours' AND tenant_id = $1", user.TenantID).Scan(&stats.RecentAlerts); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to count alerts: %v", err)
		}
	}

	// 5. Sync Status
	// Check if there are any failed syncs in last hour
	var failedCount int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM iam.sync_status WHERE status = 'failed' AND synced_at > NOW() - INTERVAL '1 hour'").Scan(&failedCount); err != nil {
		failedCount = 0
	}

	if failedCount > 0 {
		stats.SyncStatus = "degraded"
	} else {
		stats.SyncStatus = "healthy"
	}

	// 6. Last Sync Time
	var lastSync time.Time
	if err := s.DB.QueryRowContext(ctx, "SELECT MAX(synced_at) FROM iam.sync_status WHERE status = 'success'").Scan(&lastSync); err == nil {
		stats.LastSyncTime = lastSync.Format(time.RFC3339)
	} else {
		stats.LastSyncTime = ""
	}

	respond(w, r, stats, nil)
}
