package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/events"
	"github.com/hondyman/uisce/backend/internal/services"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// RegisterConnectionsRoutes registers connection management endpoints
func RegisterConnectionsRoutes(r chi.Router, db *sqlx.DB) {
	// KafkaPublisher connects lazily, so constructing it unconditionally is
	// safe even if Kafka isn't reachable in this environment — publishing a
	// gold-copy connection change is best-effort (see
	// ConnectionsService.publishGoldCopyChange).
	kafkaBrokers := getEnv("KAFKA_BROKERS", "redpanda:9092")
	connService := services.NewConnectionsServiceWithEvents(db, events.NewKafkaPublisher(kafkaBrokers))

	r.Route("/connections", func(r chi.Router) {
		// List connections for a tenant
		r.Get("/", handleListConnections(connService))
		// Create a new connection
		r.Post("/", handleCreateConnection(connService))
		// Get a specific connection
		r.Get("/{id}", handleGetConnection(connService))
		// Update a connection
		r.Put("/{id}", handleUpdateConnection(connService))
		r.Patch("/{id}", handleUpdateConnection(connService))
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

// getTenantIDFromRequest resolves the tenant to operate on. It NEVER trusts
// the client-supplied X-Tenant-ID header directly: the header is only
// honored when a valid JWT is present and that JWT grants access to the
// requested tenant (via ValidateTenantAccess, which also allows core/global
// admins to select any tenant). If JWT validation fails or the header names
// a tenant the caller isn't authorized for, this returns "" and the caller
// must reject the request rather than silently trusting an unvalidated
// value.
func getTenantIDFromRequest(r *http.Request) string {
	claims, err := jwtmiddleware.ValidateTokenFromRequest(r)
	if err != nil || claims == nil {
		return ""
	}

	headerTenant := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if headerTenant == "" {
		return claims.TenantID
	}

	if verr := jwtmiddleware.ValidateTenantAccess(claims, headerTenant); verr != nil {
		return ""
	}
	return headerTenant
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

// testConnectionByType does a shallow "is this connection populated enough
// to attempt a real connect" check. The actual network test (including
// mTLS/key_pair auth) is performed by POST /api/connections/test via
// CatalogScanService.TestConnection; this endpoint predates that and is kept
// for callers hitting /tenant-ops/connections/{id}/test directly.
func testConnectionByType(conn *services.Connection) testConnectionResult {
	switch strings.ToLower(conn.Type) {
	case "postgres", "postgresql", "mysql", "snowflake":
		return testDatabaseConnection(conn)
	case "s3":
		return testFieldConfigured(conn, conn.BaseURL != nil && *conn.BaseURL != "", "base_url")
	case "api", "rest":
		return testFieldConfigured(conn, conn.BaseURL != nil && *conn.BaseURL != "", "base_url")
	default:
		return testConnectionResult{
			Success: false,
			Message: fmt.Sprintf("connection type '%s' not supported for testing", conn.Type),
			Type:    conn.Type,
		}
	}
}

func testDatabaseConnection(conn *services.Connection) testConnectionResult {
	if conn.Host == nil || *conn.Host == "" {
		return testConnectionResult{
			Success: false,
			Message: fmt.Sprintf("host is required for %s connections", conn.Type),
			Type:    conn.Type,
		}
	}

	return testConnectionResult{
		Success: true,
		Message: fmt.Sprintf("%s connection is configured", conn.Type),
		Type:    conn.Type,
		Details: map[string]interface{}{
			"host": *conn.Host,
		},
	}
}

func testFieldConfigured(conn *services.Connection, ok bool, field string) testConnectionResult {
	if !ok {
		return testConnectionResult{
			Success: false,
			Message: fmt.Sprintf("%s is required for %s connections", field, conn.Type),
			Type:    conn.Type,
		}
	}

	return testConnectionResult{
		Success: true,
		Message: fmt.Sprintf("%s connection is configured", conn.Type),
		Type:    conn.Type,
	}
}
