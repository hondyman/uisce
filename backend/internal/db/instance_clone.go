package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
	// Trino driver temporarily disabled due to Go version compatibility issues
	// _ "github.com/trinodb/trino-go-client/trino" // Trino driver for audit logging
)

// ClonedInstance holds the result of an instance cloning operation
type ClonedInstance struct {
	InstanceID          uuid.UUID   `json:"instance_id"`
	ProductsCloned      int         `json:"products_cloned"`
	DatasourcesCloned   int         `json:"datasources_cloned"`
	ConnectionsCloned   int         `json:"connections_cloned"`
	ClonedProductIDs    []uuid.UUID `json:"cloned_product_ids"`
	ClonedDatasourceIDs []uuid.UUID `json:"cloned_datasource_ids"`
	ClonedConnectionIDs []uuid.UUID `json:"cloned_connection_ids"`
}

// ConnectionSyncResult holds the result of a connection sync operation
type ConnectionSyncResult struct {
	InstancesSynced     int         `json:"instances_synced"`
	ConnectionsCreated  int         `json:"connections_created"`
	ConnectionsUpdated  int         `json:"connections_updated"`
	ConnectionsDeleted  int         `json:"connections_deleted"`
	AffectedInstanceIDs []uuid.UUID `json:"affected_instance_ids"`
}

// GoldCopySyncAuditEntry represents an audit log entry for Gold Copy sync operations
type GoldCopySyncAuditEntry struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	GoldCopyTenantID  uuid.UUID   `json:"gold_copy_tenant_id"` // Actual Gold Copy tenant ID
	SourceEntityType  string      `json:"source_entity_type" db:"source_entity_type"`
	SourceEntityID    uuid.UUID   `json:"source_entity_id" db:"source_entity_id"`
	Operation         string      `json:"operation" db:"operation"`       // INSERT, UPDATE, DELETE
	CascadeType       string      `json:"cascade_type" db:"cascade_type"` // clone, update, delete
	AffectedCount     int         `json:"affected_count" db:"affected_count"`
	AffectedTenantIDs []uuid.UUID `json:"affected_tenant_ids"`  // For JSON storage
	AffectedEntityIDs []uuid.UUID `json:"affected_entity_ids"`  // For JSON storage
	Details           string      `json:"details" db:"details"` // JSON blob for additional context
	Timestamp         string      `json:"timestamp" db:"timestamp"`
}

// LogGoldCopySyncAudit writes an audit entry for Gold Copy sync operations to Iceberg via Trino
func LogGoldCopySyncAudit(
	ctx context.Context,
	db *sqlx.DB,
	entry GoldCopySyncAuditEntry,
) error {
	logger := logging.GetLogger().Sugar()

	// Ensure ID is set
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}

	// If GoldCopyTenantID is not provided, query for it
	if entry.GoldCopyTenantID == uuid.Nil && db != nil {
		var goldCopyTenantID uuid.UUID
		err := db.GetContext(ctx, &goldCopyTenantID, `
			SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1
		`)
		if err == nil {
			entry.GoldCopyTenantID = goldCopyTenantID
		} else {
			logger.Warnf("Failed to query Gold Copy tenant ID: %v", err)
		}
	}

	// Build details JSON with all sync metadata
	detailsMap := map[string]interface{}{
		"source_entity_type":  entry.SourceEntityType,
		"source_entity_id":    entry.SourceEntityID.String(),
		"cascade_type":        entry.CascadeType,
		"affected_count":      entry.AffectedCount,
		"affected_tenant_ids": entry.AffectedTenantIDs,
		"affected_entity_ids": entry.AffectedEntityIDs,
		"gold_copy_tenant_id": entry.GoldCopyTenantID.String(),
	}
	detailsBytes, _ := json.Marshal(detailsMap)
	detailsStr := string(detailsBytes)

	// Connect to Trino for audit logging
	trinoDSN := "http://admin@trino:8080?catalog=iceberg&schema=audit"
	trinoDB, err := sql.Open("trino", trinoDSN)
	if err != nil {
		logger.Warnf("Failed to connect to Trino for audit: %v", err)
		// Fall back to structured log only
		logger.Infof("AUDIT [Gold Copy Sync] %s %s on %s:%s -> %d entities affected across %d tenants",
			entry.Operation, entry.CascadeType, entry.SourceEntityType, entry.SourceEntityID,
			entry.AffectedCount, len(entry.AffectedTenantIDs))
		return nil
	}
	defer trinoDB.Close()

	// Determine tenant_id to use - use actual Gold Copy tenant ID if available
	tenantIDStr := entry.GoldCopyTenantID.String()
	if entry.GoldCopyTenantID == uuid.Nil {
		tenantIDStr = "00000000-0000-0000-0000-000000000000" // Fallback
	}

	// Write to Iceberg audit table
	// Using existing audit_logs table schema: id, tenant_id, timestamp, user_name, user_email, action, resource, resource_type, details
	query := `
		INSERT INTO iceberg.audit.audit_logs 
		(id, tenant_id, timestamp, user_name, user_email, action, resource, resource_type, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = trinoDB.ExecContext(ctx, query,
		entry.ID.String(),
		tenantIDStr,      // tenant_id - use actual Gold Copy tenant UUID
		time.Now().UTC(), // timestamp
		"system",         // user_name - system operation
		"",               // user_email
		entry.Operation+"_"+entry.CascadeType, // action (e.g., "DELETE_cascade_delete")
		entry.SourceEntityID.String(),         // resource
		entry.SourceEntityType,                // resource_type
		detailsStr,                            // details JSON
	)

	if err != nil {
		// Log but don't fail the operation - audit is best-effort
		logger.Warnf("Failed to write Gold Copy sync audit to Trino: %v", err)
		// Also log to stdout for observability
		logger.Infof("AUDIT [Gold Copy Sync] %s %s on %s:%s -> %d entities affected across %d tenants (Trino write failed)",
			entry.Operation, entry.CascadeType, entry.SourceEntityType, entry.SourceEntityID,
			entry.AffectedCount, len(entry.AffectedTenantIDs))
		return nil // Don't return error - audit is best-effort
	}

	logger.Infof("AUDIT [Gold Copy Sync] %s %s on %s:%s -> %d entities affected across %d tenants (logged to Iceberg)",
		entry.Operation, entry.CascadeType, entry.SourceEntityType, entry.SourceEntityID,
		entry.AffectedCount, len(entry.AffectedTenantIDs))

	return nil
}

// CloneGoldCopyInstance clones products, datasources, and connections from Gold Copy instance
// to a newly created tenant instance. All cloned items will have core_id set to reference
// the original Gold Copy item.
func CloneGoldCopyInstance(
	ctx context.Context,
	db *sqlx.DB,
	targetTenantID uuid.UUID,
	targetInstanceID uuid.UUID,
) (*ClonedInstance, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Starting Gold Copy clone for tenant %s, instance %s", targetTenantID, targetInstanceID)

	// Start transaction
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: Find Gold Copy instance
	goldInstance, err := findGoldCopyInstance(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to find gold copy instance: %w", err)
	}
	if goldInstance == nil {
		logger.Warn("No gold copy instance found, skipping clone")
		return &ClonedInstance{InstanceID: targetInstanceID}, nil
	}

	logger.Infof("Found gold copy instance: %s", goldInstance.ID)

	// Step 2: Update target instance with core_id reference
	_, err = tx.ExecContext(ctx, `
		UPDATE public.tenant_instance 
		SET core_id = $1 
		WHERE id = $2
	`, goldInstance.ID, targetInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to update instance core_id: %w", err)
	}

	result := &ClonedInstance{
		InstanceID:          targetInstanceID,
		ClonedProductIDs:    []uuid.UUID{},
		ClonedDatasourceIDs: []uuid.UUID{},
		ClonedConnectionIDs: []uuid.UUID{},
	}

	// Step 3: Clone products
	goldProducts, err := getGoldCopyProducts(ctx, tx, goldInstance.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gold copy products: %w", err)
	}

	productMapping := make(map[uuid.UUID]uuid.UUID) // gold product ID -> new product ID
	for _, gp := range goldProducts {
		newProductID := uuid.New()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_product 
				(id, datasource_id, alpha_product_id, tenant_id, version, is_active, core_id, created_at, updated_at)
			VALUES 
				($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, newProductID, targetInstanceID, gp.AlphaProductID, targetTenantID, gp.Version, false, gp.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to clone product %s: %w", gp.ID, err)
		}
		productMapping[gp.ID] = newProductID
		result.ClonedProductIDs = append(result.ClonedProductIDs, newProductID)
		result.ProductsCloned++
		logger.Infof("Cloned product %s -> %s", gp.ID, newProductID)
	}

	// Step 4: Clone connections (with blank credentials)
	goldConnections, err := getGoldCopyConnections(ctx, tx, goldInstance.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gold copy connections: %w", err)
	}

	connectionMapping := make(map[uuid.UUID]uuid.UUID) // gold connection ID -> new connection ID
	for _, gc := range goldConnections {
		newConnID, err := syncConnectionToInstance(ctx, tx, gc, targetTenantID, targetInstanceID, logger)
		if err != nil {
			return nil, err
		}
		if newConnID != uuid.Nil {
			connectionMapping[gc.ID] = newConnID
			result.ClonedConnectionIDs = append(result.ClonedConnectionIDs, newConnID)
			result.ConnectionsCloned++
		}
	}

	// Step 5: Clone datasources
	goldDatasources, err := getGoldCopyDatasources(ctx, tx, goldInstance.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gold copy datasources: %w", err)
	}

	for _, gd := range goldDatasources {
		newDatasourceID := uuid.New()
		newProductID, ok := productMapping[gd.TenantProductID]
		if !ok {
			logger.Warnf("No product mapping found for datasource %s, skipping", gd.ID)
			continue
		}

		// Map connection if exists
		var newConnectionID *uuid.UUID
		if gd.ConnectionID.Valid {
			if mappedConnID, ok := connectionMapping[gd.ConnectionID.UUID]; ok {
				newConnectionID = &mappedConnID
			}
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO public.tenant_product_datasource 
				(id, tenant_product_id, alpha_datasource_id, source_name, is_active, 
				 connection_id, datasource_id, core_id, config, created_at, updated_at)
			VALUES 
				($1, $2, $3, $4, false, $5, $6, $7, $8, NOW(), NOW())
		`, newDatasourceID, newProductID, gd.AlphaDatasourceID, gd.SourceName,
			newConnectionID, targetInstanceID, gd.ID, gd.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to clone datasource %s: %w", gd.ID, err)
		}
		result.ClonedDatasourceIDs = append(result.ClonedDatasourceIDs, newDatasourceID)
		result.DatasourcesCloned++
		logger.Infof("Cloned datasource %s -> %s", gd.ID, newDatasourceID)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Infof("Successfully cloned gold copy: %d products, %d connections, %d datasources",
		result.ProductsCloned, result.ConnectionsCloned, result.DatasourcesCloned)

	return result, nil
}

// SyncGoldCopyConnectionToAllInstances syncs a single Gold Copy connection to all non-Gold Copy instances
func SyncGoldCopyConnectionToAllInstances(
	ctx context.Context,
	db *sqlx.DB,
	goldConnectionID uuid.UUID,
	operation string,
) (*ConnectionSyncResult, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Syncing Gold Copy connection %s (op: %s) to all instances", goldConnectionID, operation)

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get the Gold Copy connection details
	var gc goldCopyConnection
	err = tx.GetContext(ctx, &gc, `
		SELECT id, name, type, host, port, database, schema, base_url, tenant_product_id, COALESCE(metadata, '{}'::jsonb) as metadata
		FROM public.connections
		WHERE id = $1
	`, goldConnectionID)
	if err != nil {
		if err == sql.ErrNoRows && operation == "DELETE" {
			// Connection already deleted, proceed with deletion sync
			gc.ID = goldConnectionID
		} else {
			return nil, fmt.Errorf("failed to get gold copy connection: %w", err)
		}
	}

	// Get all non-Gold Copy tenant instances
	var instances []struct {
		TenantID   uuid.UUID `db:"tenant_id"`
		InstanceID uuid.UUID `db:"instance_id"`
	}
	err = tx.SelectContext(ctx, &instances, `
		SELECT ti.tenant_id, ti.id as instance_id
		FROM public.tenant_instance ti
		JOIN public.tenants t ON ti.tenant_id = t.id
		WHERE t.gold_copy = false
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant instances: %w", err)
	}

	result := &ConnectionSyncResult{
		AffectedInstanceIDs: []uuid.UUID{},
	}

	for _, inst := range instances {
		if operation == "DELETE" {
			// Delete the connection from this instance
			res, err := tx.ExecContext(ctx, `
				DELETE FROM public.connections
				WHERE tenant_id = $1 AND core_id = $2
			`, inst.TenantID, goldConnectionID)
			if err != nil {
				logger.Warnf("Failed to delete connection for instance %s: %v", inst.InstanceID, err)
				continue
			}
			rows, _ := res.RowsAffected()
			if rows > 0 {
				result.ConnectionsDeleted++
				result.AffectedInstanceIDs = append(result.AffectedInstanceIDs, inst.InstanceID)
			}
		} else {
			// INSERT or UPDATE
			newConnID, err := syncConnectionToInstance(ctx, tx, gc, inst.TenantID, inst.InstanceID, logger)
			if err != nil {
				logger.Warnf("Failed to sync connection to instance %s: %v", inst.InstanceID, err)
				continue
			}
			if newConnID != uuid.Nil {
				// Check if this was an update or insert
				var existingID uuid.UUID
				err := tx.GetContext(ctx, &existingID, `
					SELECT id FROM public.connections
					WHERE tenant_id = $1 AND core_id = $2 AND id != $3
				`, inst.TenantID, goldConnectionID, newConnID)
				if err == nil {
					result.ConnectionsUpdated++
				} else {
					result.ConnectionsCreated++
				}
				result.AffectedInstanceIDs = append(result.AffectedInstanceIDs, inst.InstanceID)
			}
		}
		result.InstancesSynced++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Infof("Synced connection to %d instances (%d created, %d updated, %d deleted)",
		result.InstancesSynced, result.ConnectionsCreated, result.ConnectionsUpdated, result.ConnectionsDeleted)

	// Audit log the sync operation
	_ = LogGoldCopySyncAudit(ctx, db, GoldCopySyncAuditEntry{
		SourceEntityType:  "connection",
		SourceEntityID:    goldConnectionID,
		Operation:         operation,
		CascadeType:       "sync",
		AffectedCount:     result.ConnectionsCreated + result.ConnectionsUpdated + result.ConnectionsDeleted,
		AffectedTenantIDs: result.AffectedInstanceIDs, // Instance IDs represent tenants here
	})

	return result, nil
}

// SyncAllConnectionsForInstance syncs all Gold Copy connections to a specific instance
func SyncAllConnectionsForInstance(
	ctx context.Context,
	db *sqlx.DB,
	targetTenantID uuid.UUID,
	targetInstanceID uuid.UUID,
) (*ConnectionSyncResult, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Syncing all Gold Copy connections to instance %s (tenant %s)", targetInstanceID, targetTenantID)

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Find Gold Copy tenant
	goldInstance, err := findGoldCopyInstance(ctx, tx)
	if err != nil {
		logger.Errorf("Error finding gold copy instance: %v", err)
		return nil, fmt.Errorf("failed to find gold copy instance: %w", err)
	}
	if goldInstance == nil {
		logger.Warn("No gold copy instance found in database")
		return nil, fmt.Errorf("no gold copy instance found")
	}

	logger.Infof("Found Gold Copy instance: %s (tenant: %s)", goldInstance.ID, goldInstance.TenantID)

	// Get all Gold Copy connections
	goldConnections, err := getGoldCopyConnections(ctx, tx, goldInstance.TenantID)
	if err != nil {
		logger.Errorf("Error getting gold copy connections: %v", err)
		return nil, fmt.Errorf("failed to get gold copy connections: %w", err)
	}

	logger.Infof("Found %d connections in Gold Copy tenant", len(goldConnections))

	result := &ConnectionSyncResult{
		AffectedInstanceIDs: []uuid.UUID{targetInstanceID},
	}

	for _, gc := range goldConnections {
		logger.Infof("Syncing connection: %s (%s)", gc.Name, gc.ID)
		newConnID, err := syncConnectionToInstance(ctx, tx, gc, targetTenantID, targetInstanceID, logger)
		if err != nil {
			logger.Warnf("Failed to sync connection %s: %v", gc.ID, err)
			continue
		}
		if newConnID != uuid.Nil {
			// Check if this was an update or insert
			var existingID uuid.UUID
			err := tx.GetContext(ctx, &existingID, `
				SELECT id FROM public.connections
				WHERE tenant_id = $1 AND core_id = $2 AND id != $3
			`, targetTenantID, gc.ID, newConnID)
			if err == nil {
				result.ConnectionsUpdated++
				logger.Infof("Updated existing connection %s", newConnID)
			} else {
				result.ConnectionsCreated++
				logger.Infof("Created new connection %s", newConnID)
			}

			// Link relevant datasources to this connection
			if err := linkDatasourcesToConnection(ctx, tx, gc.ID, newConnID, targetInstanceID, logger); err != nil {
				logger.Warnf("Failed to link datasources for connection %s: %v", newConnID, err)
				// processing continues even if linking fails
			}
		}
	}

	result.InstancesSynced = 1

	if err := tx.Commit(); err != nil {
		logger.Errorf("Failed to commit transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Infof("Synced %d connections to instance (%d created, %d updated)",
		result.ConnectionsCreated+result.ConnectionsUpdated, result.ConnectionsCreated, result.ConnectionsUpdated)

	return result, nil
}

// linkDatasourcesToConnection updates datasources in the target instance to point to the new connection
// if the corresponding Gold Copy datasource uses the Gold Copy connection.
func linkDatasourcesToConnection(
	ctx context.Context,
	tx *sqlx.Tx,
	goldConnectionID uuid.UUID,
	targetConnectionID uuid.UUID,
	targetInstanceID uuid.UUID,
	logger interface{ Infof(string, ...interface{}) },
) error {
	// Update all datasources in the target instance that:
	// 1. Belong to the target instance
	// 2. Have a core_id that matches a Gold Copy datasource
	// 3. That Gold Copy datasource is linked to the Gold Copy connection
	result, err := tx.ExecContext(ctx, `
		UPDATE tenant_product_datasource tpd
		SET connection_id = $1, updated_at = NOW()
		WHERE tpd.datasource_id = $2
		  AND tpd.core_id IN (
			  SELECT gc_tpd.id 
			  FROM tenant_product_datasource gc_tpd
			  WHERE gc_tpd.connection_id = $3
		  )
	`, targetConnectionID, targetInstanceID, goldConnectionID)

	if err != nil {
		return fmt.Errorf("failed to link datasources: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		logger.Infof("Linked %d datasources to connection %s", rows, targetConnectionID)
	}

	return nil
}

// syncConnectionToInstance is a helper that syncs a single connection to an instance
// Returns the connection ID (new or existing) or uuid.Nil if skipped
func syncConnectionToInstance(
	ctx context.Context,
	tx *sqlx.Tx,
	gc goldCopyConnection,
	targetTenantID uuid.UUID,
	targetInstanceID uuid.UUID,
	logger interface{ Infof(string, ...interface{}) },
) (uuid.UUID, error) {
	// Products are NOT cloned - all tenants share the same Gold Copy products
	// So we directly use the Gold Copy's tenant_product_id
	targetProductID := gc.TenantProductID

	// Check if a connection for this core already exists for this tenant
	var existingConnectionID uuid.UUID
	err := tx.GetContext(ctx, &existingConnectionID, `
		SELECT id FROM public.connections 
		WHERE tenant_id = $1 AND core_id = $2 AND datasource_id = $3
	`, targetTenantID, gc.ID, targetInstanceID)

	if err == nil {
		// Connection already exists, update it
		// Sanitize metadata
		sanitizedMetadata, err := sanitizeConnectionMetadata(gc.Metadata)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to sanitize metadata: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE public.connections
			SET name = $1, type = $2, schema = $3, port = $4, metadata = $5, datasource_id = $6, tenant_product_id = $7, updated_at = NOW()
			WHERE id = $8
		`, gc.Name, gc.Type, gc.Schema, gc.Port, sanitizedMetadata, targetInstanceID, targetProductID, existingConnectionID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to update connection: %w", err)
		}
		logger.Infof("Updated connection %s for instance %s", existingConnectionID, targetInstanceID)
		return existingConnectionID, nil
	}

	// Connection doesn't exist, create it
	newConnectionID := uuid.New()

	// Sanitize metadata to remove auth_type and sensitive keys
	sanitizedMetadata, err := sanitizeConnectionMetadata(gc.Metadata)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to sanitize metadata: %w", err)
	}

	// Clone connection with blank credentials and inactive status
	// Host, Database, BaseURL, AuthType (in metadata), Username, Password must be populated independently
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.connections 
			(id, tenant_id, datasource_id, name, type, host, port, database, schema, username, password, 
			 api_key, base_url, metadata, is_active, core_id, tenant_product_id, created_at, updated_at)
		VALUES 
			($1, $2, $3, $4, $5, NULL, $6, NULL, $7, NULL, NULL, NULL, NULL, $8, false, $9, $10, NOW(), NOW())
	`, newConnectionID, targetTenantID, targetInstanceID, gc.Name, gc.Type, gc.Port, gc.Schema,
		sanitizedMetadata, gc.ID, targetProductID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to clone connection %s: %w", gc.ID, err)
	}
	logger.Infof("Cloned connection %s -> %s (credentials and host/db/auth cleared)", gc.ID, newConnectionID)
	return newConnectionID, nil
}

// sanitizeConnectionMetadata removes sensitive fields from metadata
func sanitizeConnectionMetadata(metadata []byte) ([]byte, error) {
	var metaMap map[string]interface{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &metaMap); err != nil {
			// If invalid JSON, start with empty map
			metaMap = make(map[string]interface{})
		}
	} else {
		metaMap = make(map[string]interface{})
	}

	// Remove fields that should be populated independently per instance
	delete(metaMap, "auth_type")
	delete(metaMap, "api_key")
	delete(metaMap, "base_url")

	return json.Marshal(metaMap)
}

// Helper types and functions

type goldCopyInstanceInfo struct {
	ID       uuid.UUID `db:"id"`
	TenantID uuid.UUID `db:"tenant_id"`
}

type goldCopyProduct struct {
	ID             uuid.UUID `db:"id"`
	AlphaProductID uuid.UUID `db:"alpha_product_id"`
	Version        float64   `db:"version"`
}

type goldCopyConnection struct {
	ID              uuid.UUID      `db:"id"`
	Name            string         `db:"name"`
	Type            string         `db:"type"`
	Host            sql.NullString `db:"host"`
	Port            sql.NullInt32  `db:"port"`
	Database        sql.NullString `db:"database"`
	Schema          sql.NullString `db:"schema"`
	BaseURL         sql.NullString `db:"base_url"`
	TenantProductID uuid.NullUUID  `db:"tenant_product_id"`
	Metadata        []byte         `db:"metadata"`
}

type goldCopyDatasource struct {
	ID                uuid.UUID      `db:"id"`
	TenantProductID   uuid.UUID      `db:"tenant_product_id"`
	AlphaDatasourceID uuid.UUID      `db:"alpha_datasource_id"`
	SourceName        sql.NullString `db:"source_name"`
	ConnectionID      uuid.NullUUID  `db:"connection_id"`
	Config            []byte         `db:"config"`
}

func findGoldCopyInstance(ctx context.Context, tx *sqlx.Tx) (*goldCopyInstanceInfo, error) {
	var info goldCopyInstanceInfo
	err := tx.GetContext(ctx, &info, `
		SELECT ti.id, ti.tenant_id
		FROM public.tenant_instance ti
		JOIN public.tenants t ON ti.tenant_id = t.id
		WHERE t.gold_copy = true
		LIMIT 1
	`)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func getGoldCopyProducts(ctx context.Context, tx *sqlx.Tx, instanceID uuid.UUID) ([]goldCopyProduct, error) {
	var products []goldCopyProduct
	err := tx.SelectContext(ctx, &products, `
		SELECT id, alpha_product_id, version
		FROM public.tenant_product
		WHERE datasource_id = $1
	`, instanceID)
	return products, err
}

func getGoldCopyConnections(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID) ([]goldCopyConnection, error) {
	var connections []goldCopyConnection
	err := tx.SelectContext(ctx, &connections, `
		SELECT id, name, type, host, port, database, schema, base_url, tenant_product_id, COALESCE(metadata, '{}'::jsonb) as metadata
		FROM public.connections
		WHERE tenant_id = $1
	`, tenantID)
	return connections, err
}

func getGoldCopyDatasources(ctx context.Context, tx *sqlx.Tx, instanceID uuid.UUID) ([]goldCopyDatasource, error) {
	var datasources []goldCopyDatasource
	err := tx.SelectContext(ctx, &datasources, `
		SELECT tpd.id, tpd.tenant_product_id, tpd.alpha_datasource_id, tpd.source_name, 
		       tpd.connection_id, COALESCE(tpd.config, '{}'::jsonb) as config
		FROM public.tenant_product_datasource tpd
		JOIN public.tenant_product tp ON tpd.tenant_product_id = tp.id
		WHERE tp.datasource_id = $1
	`, instanceID)
	return datasources, err
}

// EntitySyncResult holds the result of a sync deletion operation
type EntitySyncResult struct {
	EntitiesDeleted int         `json:"entities_deleted"`
	AffectedTenants []uuid.UUID `json:"affected_tenants"`
}

// SyncGoldCopyInstanceDeletion removes cloned instances across all non-Gold Copy tenants
// when the Gold Copy instance is deleted. Uses core_id to find clones.
func SyncGoldCopyInstanceDeletion(
	ctx context.Context,
	db *sqlx.DB,
	goldInstanceID uuid.UUID,
) (*EntitySyncResult, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Syncing Gold Copy instance deletion: %s", goldInstanceID)

	// Find all instances that were cloned from this Gold Copy instance
	var clonedInstances []struct {
		ID       uuid.UUID `db:"id"`
		TenantID uuid.UUID `db:"tenant_id"`
	}
	err := db.SelectContext(ctx, &clonedInstances, `
		SELECT id, tenant_id
		FROM public.tenant_instance
		WHERE core_id = $1
	`, goldInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find cloned instances: %w", err)
	}

	if len(clonedInstances) == 0 {
		logger.Info("No cloned instances found to delete")
		return &EntitySyncResult{EntitiesDeleted: 0, AffectedTenants: []uuid.UUID{}}, nil
	}

	// Delete cloned instances (cascading FK constraints will handle related data)
	result := &EntitySyncResult{
		AffectedTenants: []uuid.UUID{},
	}

	for _, inst := range clonedInstances {
		_, err := db.ExecContext(ctx, `
			DELETE FROM public.tenant_instance WHERE id = $1
		`, inst.ID)
		if err != nil {
			logger.Warnf("Failed to delete cloned instance %s: %v", inst.ID, err)
			continue
		}
		result.EntitiesDeleted++
		result.AffectedTenants = append(result.AffectedTenants, inst.TenantID)
	}

	logger.Infof("Deleted %d cloned instances across %d tenants", result.EntitiesDeleted, len(result.AffectedTenants))

	// Audit log the deletion cascade
	_ = LogGoldCopySyncAudit(ctx, db, GoldCopySyncAuditEntry{
		SourceEntityType:  "tenant_instance",
		SourceEntityID:    goldInstanceID,
		Operation:         "DELETE",
		CascadeType:       "cascade_delete",
		AffectedCount:     result.EntitiesDeleted,
		AffectedTenantIDs: result.AffectedTenants,
	})

	return result, nil
}

// SyncGoldCopyProductDeletion removes cloned products across all non-Gold Copy tenants
// when the Gold Copy product is deleted. Uses core_id to find clones.
func SyncGoldCopyProductDeletion(
	ctx context.Context,
	db *sqlx.DB,
	goldProductID uuid.UUID,
) (*EntitySyncResult, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Syncing Gold Copy product deletion: %s", goldProductID)

	// Find all products that were cloned from this Gold Copy product
	var clonedProducts []struct {
		ID       uuid.UUID `db:"id"`
		TenantID uuid.UUID `db:"tenant_id"`
	}
	err := db.SelectContext(ctx, &clonedProducts, `
		SELECT id, tenant_id
		FROM public.tenant_product
		WHERE core_id = $1
	`, goldProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to find cloned products: %w", err)
	}

	if len(clonedProducts) == 0 {
		logger.Info("No cloned products found to delete")
		return &EntitySyncResult{EntitiesDeleted: 0, AffectedTenants: []uuid.UUID{}}, nil
	}

	// Delete cloned products (cascading FK constraints will handle related datasources)
	result := &EntitySyncResult{
		AffectedTenants: []uuid.UUID{},
	}

	for _, prod := range clonedProducts {
		_, err := db.ExecContext(ctx, `
			DELETE FROM public.tenant_product WHERE id = $1
		`, prod.ID)
		if err != nil {
			logger.Warnf("Failed to delete cloned product %s: %v", prod.ID, err)
			continue
		}
		result.EntitiesDeleted++
		result.AffectedTenants = append(result.AffectedTenants, prod.TenantID)
	}

	logger.Infof("Deleted %d cloned products across %d tenants", result.EntitiesDeleted, len(result.AffectedTenants))

	// Audit log the deletion cascade
	_ = LogGoldCopySyncAudit(ctx, db, GoldCopySyncAuditEntry{
		SourceEntityType:  "tenant_product",
		SourceEntityID:    goldProductID,
		Operation:         "DELETE",
		CascadeType:       "cascade_delete",
		AffectedCount:     result.EntitiesDeleted,
		AffectedTenantIDs: result.AffectedTenants,
	})

	return result, nil
}
