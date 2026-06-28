package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// RegisterConnectionsRoutes registers connection management endpoints
func RegisterConnectionsRoutes(r chi.Router, db *sqlx.DB) {
	connService := services.NewConnectionsService(db)

	r.Route("/connections", func(r chi.Router) {
		// List connections for a tenant
		r.Get("/", handleListConnections(connService))
		// Create a new connection
		r.Post("/", handleCreateConnection(connService))
		// Get a specific connection
		r.Get("/{id}", handleGetConnection(connService))
		// Update a connection
		r.Put("/{id}", handleUpdateConnection(connService))
		// Delete a connection
		r.Delete("/{id}", handleDeleteConnection(connService))
		// Link connection to datasource
		r.Post("/{id}/link/{datasourceId}", handleLinkConnection(connService))
		// Unlink connection from datasource
		r.Delete("/{id}/unlink/{datasourceId}", handleUnlinkConnection(connService))
		// Get datasources for a connection
		r.Get("/{id}/datasources", handleGetConnectionDatasources(connService))
		// Test a connection
		r.Post("/{id}/test", handleTestConnection(connService))
	})
}

func getTenantIDFromRequest(r *http.Request) string {
	// Check headers first (added by tenant fetch shim)
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims != nil && claims.TenantID != "" {
		return claims.TenantID
	}
	// Check query parameters
	return r.URL.Query().Get("tenant_id")
}

// handleListConnections lists all connections for a tenant
func handleListConnections(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connections, err := svc.ListConnections(r.Context(), tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list connections", "list_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"connections": connections,
			"count":       len(connections),
		})
	}
}

// handleCreateConnection creates a new connection
func handleCreateConnection(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		var conn services.Connection
		if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body", "decode_error", err.Error())
			return
		}

		created, err := svc.CreateConnection(r.Context(), tenantID, &conn)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create connection", "create_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}
}

// handleGetConnection retrieves a specific connection
func handleGetConnection(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connID := chi.URLParam(r, "id")
		if connID == "" {
			writeJSONError(w, http.StatusBadRequest, "connection id is required", "missing_id", nil)
			return
		}

		conn, err := svc.GetConnection(r.Context(), tenantID, connID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSONError(w, http.StatusNotFound, "connection not found", "not_found", nil)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to get connection", "get_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conn)
	}
}

// handleUpdateConnection updates an existing connection
func handleUpdateConnection(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connID := chi.URLParam(r, "id")
		if connID == "" {
			writeJSONError(w, http.StatusBadRequest, "connection id is required", "missing_id", nil)
			return
		}

		var conn services.Connection
		if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body", "decode_error", err.Error())
			return
		}

		conn.ID = connID
		updated, err := svc.UpdateConnection(r.Context(), tenantID, &conn)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSONError(w, http.StatusNotFound, "connection not found", "not_found", nil)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to update connection", "update_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updated)
	}
}

// handleDeleteConnection deletes a connection
func handleDeleteConnection(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connID := chi.URLParam(r, "id")
		if connID == "" {
			writeJSONError(w, http.StatusBadRequest, "connection id is required", "missing_id", nil)
			return
		}

		err := svc.DeleteConnection(r.Context(), tenantID, connID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSONError(w, http.StatusNotFound, "connection not found", "not_found", nil)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to delete connection", "delete_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "connection deleted successfully",
		})
	}
}

// handleLinkConnection links a connection to a datasource
func handleLinkConnection(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connID := chi.URLParam(r, "id")
		datasourceID := chi.URLParam(r, "datasourceId")

		if connID == "" || datasourceID == "" {
			writeJSONError(w, http.StatusBadRequest, "connection and datasource ids are required", "missing_ids", nil)
			return
		}

		err := svc.LinkConnectionToDatasource(r.Context(), tenantID, datasourceID, connID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSONError(w, http.StatusNotFound, "datasource not found", "not_found", nil)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to link connection", "link_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"message":       "connection linked to datasource",
			"connection_id": connID,
			"datasource_id": datasourceID,
		})
	}
}

// handleUnlinkConnection unlinks a connection from a datasource
func handleUnlinkConnection(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connID := chi.URLParam(r, "id")
		datasourceID := chi.URLParam(r, "datasourceId")

		if connID == "" || datasourceID == "" {
			writeJSONError(w, http.StatusBadRequest, "connection and datasource ids are required", "missing_ids", nil)
			return
		}

		err := svc.UnlinkConnectionFromDatasource(r.Context(), tenantID, datasourceID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSONError(w, http.StatusNotFound, "datasource not found", "not_found", nil)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to unlink connection", "unlink_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       true,
			"message":       "connection unlinked from datasource",
			"datasource_id": datasourceID,
		})
	}
}

// handleGetConnectionDatasources retrieves all datasources for a connection
func handleGetConnectionDatasources(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connID := chi.URLParam(r, "id")
		if connID == "" {
			writeJSONError(w, http.StatusBadRequest, "connection id is required", "missing_id", nil)
			return
		}

		datasources, err := svc.GetDatasourcesForConnection(r.Context(), tenantID, connID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to get datasources", "get_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"connection_id": connID,
			"datasources":   datasources,
			"count":         len(datasources),
		})
	}
}

// handleTestConnection tests a connection's validity
func handleTestConnection(svc *services.ConnectionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantIDFromRequest(r)
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}

		connID := chi.URLParam(r, "id")
		if connID == "" {
			writeJSONError(w, http.StatusBadRequest, "connection id is required", "missing_id", nil)
			return
		}

		conn, err := svc.GetConnection(r.Context(), tenantID, connID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeJSONError(w, http.StatusNotFound, "connection not found", "not_found", nil)
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to get connection", "get_error", err.Error())
			return
		}

		// Test connection based on type
		testResult := testConnectionByType(conn)

		w.Header().Set("Content-Type", "application/json")
		if testResult.Success {
			json.NewEncoder(w).Encode(testResult)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(testResult)
		}
	}
}

type testConnectionResult struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Details interface{} `json:"details,omitempty"`
}

// testConnectionByType tests a connection based on its type
func testConnectionByType(conn *services.Connection) testConnectionResult {
	switch strings.ToLower(conn.Type) {
	case "postgres", "postgresql":
		return testPostgresConnection(conn)
	case "mysql":
		return testMySQLConnection(conn)
	case "snowflake":
		return testSnowflakeConnection(conn)
	case "s3":
		return testS3Connection(conn)
	case "api", "rest":
		return testAPIConnection(conn)
	default:
		return testConnectionResult{
			Success: false,
			Message: fmt.Sprintf("connection type '%s' not supported for testing", conn.Type),
			Type:    conn.Type,
		}
	}
}

func testPostgresConnection(conn *services.Connection) testConnectionResult {
	if conn.Host == nil || conn.Port == nil || conn.Database == nil {
		return testConnectionResult{
			Success: false,
			Message: "host, port, and database are required for postgres connections",
			Type:    "postgres",
		}
	}

	// In a real implementation, you would attempt to connect to the database
	// For now, we'll just validate the configuration
	return testConnectionResult{
		Success: true,
		Message: "postgres connection configuration is valid",
		Type:    "postgres",
		Details: map[string]interface{}{
			"host":     *conn.Host,
			"port":     *conn.Port,
			"database": *conn.Database,
		},
	}
}

func testMySQLConnection(conn *services.Connection) testConnectionResult {
	if conn.Host == nil || conn.Port == nil || conn.Database == nil {
		return testConnectionResult{
			Success: false,
			Message: "host, port, and database are required for mysql connections",
			Type:    "mysql",
		}
	}

	return testConnectionResult{
		Success: true,
		Message: "mysql connection configuration is valid",
		Type:    "mysql",
	}
}

func testSnowflakeConnection(conn *services.Connection) testConnectionResult {
	if conn.BaseURL == nil && conn.Username == nil {
		return testConnectionResult{
			Success: false,
			Message: "base_url and username are required for snowflake connections",
			Type:    "snowflake",
		}
	}

	return testConnectionResult{
		Success: true,
		Message: "snowflake connection configuration is valid",
		Type:    "snowflake",
	}
}

func testS3Connection(conn *services.Connection) testConnectionResult {
	if conn.APIKey == nil {
		return testConnectionResult{
			Success: false,
			Message: "api_key (AWS access key) is required for s3 connections",
			Type:    "s3",
		}
	}

	return testConnectionResult{
		Success: true,
		Message: "s3 connection configuration is valid",
		Type:    "s3",
	}
}

func testAPIConnection(conn *services.Connection) testConnectionResult {
	if conn.BaseURL == nil {
		return testConnectionResult{
			Success: false,
			Message: "base_url is required for api connections",
			Type:    "api",
		}
	}

	return testConnectionResult{
		Success: true,
		Message: "api connection configuration is valid",
		Type:    "api",
	}
}
