package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ConnectionSyncHandler struct {
	DB *sql.DB
}

type SyncConnectionsResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	ConnectionsAdded int    `json:"connections_added"`
	InstancesUpdated int    `json:"instances_updated"`
}

// SyncConnectionsFromGoldCopy clones connections from the gold copy tenant
// for all products registered to the target tenant
func (h *ConnectionSyncHandler) SyncConnectionsFromGoldCopy(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	if tenantID == "" {
		http.Error(w, "tenant ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// 1. Verify tenant exists and is not gold copy
	var isGoldCopy bool
	err := h.DB.QueryRowContext(ctx, `
		SELECT COALESCE(gold_copy, false) FROM tenants WHERE id = $1
	`, tenantID).Scan(&isGoldCopy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch tenant: %v", err), http.StatusInternalServerError)
		return
	}
	if isGoldCopy {
		http.Error(w, "Cannot sync connections for gold copy tenant", http.StatusBadRequest)
		return
	}

	// 2. Find the gold copy tenant
	var goldCopyTenantID string
	err = h.DB.QueryRowContext(ctx, `
		SELECT id FROM tenants WHERE gold_copy = true LIMIT 1
	`).Scan(&goldCopyTenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to find gold copy tenant: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Get tenant's registered products (alpha_product_ids)
	rows, err := h.DB.QueryContext(ctx, `
		SELECT DISTINCT alpha_product_id 
		FROM tenant_product 
		WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch tenant products: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var productIDs []string
	for rows.Next() {
		var productID string
		if err := rows.Scan(&productID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan product ID: %v", err), http.StatusInternalServerError)
			return
		}
		productIDs = append(productIDs, productID)
	}

	if len(productIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SyncConnectionsResponse{
			Success:          true,
			Message:          "No products registered to tenant",
			ConnectionsAdded: 0,
			InstancesUpdated: 0,
		})
		return
	}

	// 4. Get tenant's instances
	instanceRows, err := h.DB.QueryContext(ctx, `
		SELECT id FROM tenant_instance WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch tenant instances: %v", err), http.StatusInternalServerError)
		return
	}
	defer instanceRows.Close()

	var instanceIDs []string
	for instanceRows.Next() {
		var instanceID string
		if err := instanceRows.Scan(&instanceID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan instance ID: %v", err), http.StatusInternalServerError)
			return
		}
		instanceIDs = append(instanceIDs, instanceID)
	}

	if len(instanceIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SyncConnectionsResponse{
			Success:          true,
			Message:          "No instances found for tenant",
			ConnectionsAdded: 0,
			InstancesUpdated: 0,
		})
		return
	}

	// 5. For each product, find gold copy connections and clone them
	connectionsAdded := 0
	instancesUpdated := 0

	for _, productID := range productIDs {
		// Get gold copy connections for this product
		connRows, err := h.DB.QueryContext(ctx, `
			SELECT DISTINCT c.id, c.name, c.type, c.host, c.port, c.database, c.schema, 
			       c.username, c.password, c.base_url, c.api_key, c.metadata, c.is_active,
			       tpd.alpha_datasource_id, tpd.source_name
			FROM connections c
			JOIN tenant_product_datasource tpd ON tpd.connection_id = c.id
			JOIN tenant_product tp ON tp.id = tpd.tenant_product_id
			WHERE c.tenant_id = $1 
			  AND tp.alpha_product_id = $2
			  AND c.core_id IS NULL
		`, goldCopyTenantID, productID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch gold copy connections: %v", err), http.StatusInternalServerError)
			return
		}

		type GoldCopyConnection struct {
			ID                string
			Name              string
			Type              string
			Host              sql.NullString
			Port              sql.NullInt64
			Database          sql.NullString
			Schema            sql.NullString
			Username          sql.NullString
			Password          sql.NullString
			BaseURL           sql.NullString
			APIKey            sql.NullString
			Metadata          sql.NullString
			IsActive          bool
			AlphaDatasourceID string
			SourceName        string
		}

		var goldConnections []GoldCopyConnection
		for connRows.Next() {
			var gc GoldCopyConnection
			if err := connRows.Scan(&gc.ID, &gc.Name, &gc.Type, &gc.Host, &gc.Port, &gc.Database,
				&gc.Schema, &gc.Username, &gc.Password, &gc.BaseURL, &gc.APIKey, &gc.Metadata,
				&gc.IsActive, &gc.AlphaDatasourceID, &gc.SourceName); err != nil {
				connRows.Close()
				http.Error(w, fmt.Sprintf("Failed to scan gold copy connection: %v", err), http.StatusInternalServerError)
				return
			}
			goldConnections = append(goldConnections, gc)
		}
		connRows.Close()

		// For each gold copy connection, clone it for each instance
		for _, gc := range goldConnections {
			for _, instanceID := range instanceIDs {
				// Check if connection already exists (by core_id)
				var existingConnID string
				err := h.DB.QueryRowContext(ctx, `
					SELECT c.id 
					FROM connections c
					WHERE c.tenant_id = $1 
					  AND c.core_id = $2
					LIMIT 1
				`, tenantID, gc.ID).Scan(&existingConnID)

				if err == sql.ErrNoRows {
					// Connection doesn't exist, create it
					newConnID := uuid.New().String()
					_, err := h.DB.ExecContext(ctx, `
						INSERT INTO connections (
							id, tenant_id, name, type, host, port, database, schema,
							username, password, base_url, api_key, metadata, is_active, core_id
						) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
					`, newConnID, tenantID, gc.Name, gc.Type, gc.Host, gc.Port, gc.Database,
						gc.Schema, gc.Username, gc.Password, gc.BaseURL, gc.APIKey, gc.Metadata,
						gc.IsActive, gc.ID)
					if err != nil {
						http.Error(w, fmt.Sprintf("Failed to create connection: %v", err), http.StatusInternalServerError)
						return
					}

					// Link connection to instance via tenant_product_datasource
					// First, get the tenant_product_id for this instance and product
					var tenantProductID string
					err = h.DB.QueryRowContext(ctx, `
						SELECT id FROM tenant_product 
						WHERE datasource_id = $1 AND alpha_product_id = $2
						LIMIT 1
					`, instanceID, productID).Scan(&tenantProductID)
					if err != nil {
						// If tenant_product doesn't exist, create it
						tenantProductID = uuid.New().String()
						_, err = h.DB.ExecContext(ctx, `
							INSERT INTO tenant_product (id, datasource_id, alpha_product_id, version, is_active)
							VALUES ($1, $2, $3, 1.0, true)
						`, tenantProductID, instanceID, productID)
						if err != nil {
							http.Error(w, fmt.Sprintf("Failed to create tenant_product: %v", err), http.StatusInternalServerError)
							return
						}
					}

					// Create tenant_product_datasource link
					_, err = h.DB.ExecContext(ctx, `
						INSERT INTO tenant_product_datasource (
							tenant_product_id, datasource_id, alpha_datasource_id, 
							connection_id, source_name, is_active, config, core_id
						) VALUES ($1, $2, $3, $4, $5, true, '{}', $6)
						ON CONFLICT (tenant_product_id, connection_id) DO NOTHING
					`, tenantProductID, instanceID, gc.AlphaDatasourceID, newConnID, gc.SourceName, gc.ID)
					if err != nil {
						http.Error(w, fmt.Sprintf("Failed to create tenant_product_datasource: %v", err), http.StatusInternalServerError)
						return
					}

					connectionsAdded++
					instancesUpdated++
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SyncConnectionsResponse{
		Success:          true,
		Message:          fmt.Sprintf("Successfully synced %d connections across %d instances", connectionsAdded, instancesUpdated),
		ConnectionsAdded: connectionsAdded,
		InstancesUpdated: instancesUpdated,
	})
}
