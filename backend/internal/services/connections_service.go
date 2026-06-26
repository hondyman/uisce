package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Connection represents a unified datasource connection
type Connection struct {
	ID        string                 `json:"id" db:"id"`
	TenantID  string                 `json:"tenant_id" db:"tenant_id"`
	Name      string                 `json:"name" db:"name"`
	Type      string                 `json:"type" db:"type"` // postgres, mysql, snowflake, s3, etc.
	Host      *string                `json:"host,omitempty" db:"host"`
	Port      *int                   `json:"port,omitempty" db:"port"`
	Database  *string                `json:"database,omitempty" db:"database"`
	Schema    *string                `json:"schema,omitempty" db:"schema"`
	Username  *string                `json:"username,omitempty" db:"username"`
	Password  *string                `json:"password,omitempty" db:"password"`
	BaseURL   *string                `json:"base_url,omitempty" db:"base_url"`
	APIKey    *string                `json:"api_key,omitempty" db:"api_key"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	IsActive  bool                   `json:"is_active" db:"is_active"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// ConnectionsService handles unified connection management
type ConnectionsService struct {
	db *sqlx.DB
}

// NewConnectionsService creates a new connections service
func NewConnectionsService(db *sqlx.DB) *ConnectionsService {
	return &ConnectionsService{db: db}
}

// CreateConnection creates a new connection
func (s *ConnectionsService) CreateConnection(ctx context.Context, tenantID string, conn *Connection) (*Connection, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection cannot be nil")
	}

	conn.ID = uuid.NewString()
	conn.TenantID = tenantID
	conn.CreatedAt = time.Now()
	conn.UpdatedAt = time.Now()
	if conn.IsActive == false && conn.IsActive != true {
		conn.IsActive = true // default to active
	}

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if conn.Metadata != nil {
		metadataJSON, err = json.Marshal(conn.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO tenant_connections 
		(id, tenant_id, name, type, host, port, database, schema, username, password, base_url, api_key, metadata, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err = s.db.ExecContext(ctx, query,
		conn.ID, conn.TenantID, conn.Name, conn.Type, conn.Host, conn.Port,
		conn.Database, conn.Schema, conn.Username, conn.Password,
		conn.BaseURL, conn.APIKey, metadataJSON, conn.IsActive,
		conn.CreatedAt, conn.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	return conn, nil
}

// GetConnection retrieves a connection by ID
func (s *ConnectionsService) GetConnection(ctx context.Context, tenantID, connectionID string) (*Connection, error) {
	conn := &Connection{}
	var metadataJSON []byte

	query := `
		SELECT id, tenant_id, name, type, host, port, database, schema, username, password, base_url, api_key, metadata, is_active, created_at, updated_at
		FROM tenant_connections
		WHERE id = $1 AND tenant_id = $2
	`

	err := s.db.QueryRowContext(ctx, query, connectionID, tenantID).Scan(
		&conn.ID, &conn.TenantID, &conn.Name, &conn.Type, &conn.Host, &conn.Port,
		&conn.Database, &conn.Schema, &conn.Username, &conn.Password,
		&conn.BaseURL, &conn.APIKey, &metadataJSON, &conn.IsActive,
		&conn.CreatedAt, &conn.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection not found")
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Parse metadata
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &conn.Metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	return conn, nil
}

// ListConnections retrieves all connections for a tenant
func (s *ConnectionsService) ListConnections(ctx context.Context, tenantID string) ([]*Connection, error) {
	query := `
		SELECT id, tenant_id, name, type, host, port, database, schema, username, password, base_url, api_key, metadata, is_active, created_at, updated_at
		FROM tenant_connections
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}
	defer rows.Close()

	var connections []*Connection
	for rows.Next() {
		conn := &Connection{}
		var metadataJSON []byte

		err := rows.Scan(
			&conn.ID, &conn.TenantID, &conn.Name, &conn.Type, &conn.Host, &conn.Port,
			&conn.Database, &conn.Schema, &conn.Username, &conn.Password,
			&conn.BaseURL, &conn.APIKey, &metadataJSON, &conn.IsActive,
			&conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &conn.Metadata); err != nil {
				return nil, fmt.Errorf("failed to parse metadata: %w", err)
			}
		}

		connections = append(connections, conn)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connections: %w", err)
	}

	return connections, nil
}

// UpdateConnection updates an existing connection
func (s *ConnectionsService) UpdateConnection(ctx context.Context, tenantID string, conn *Connection) (*Connection, error) {
	if conn == nil || conn.ID == "" {
		return nil, fmt.Errorf("connection and ID cannot be nil/empty")
	}

	conn.UpdatedAt = time.Now()

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if conn.Metadata != nil {
		metadataJSON, err = json.Marshal(conn.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		UPDATE tenant_connections
		SET name = $1, type = $2, host = $3, port = $4, database = $5, schema = $6,
		    username = $7, password = $8, base_url = $9, api_key = $10, metadata = $11,
		    is_active = $12, updated_at = $13
		WHERE id = $14 AND tenant_id = $15
	`

	result, err := s.db.ExecContext(ctx, query,
		conn.Name, conn.Type, conn.Host, conn.Port,
		conn.Database, conn.Schema, conn.Username, conn.Password,
		conn.BaseURL, conn.APIKey, metadataJSON,
		conn.IsActive, conn.UpdatedAt,
		conn.ID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update connection: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return nil, fmt.Errorf("connection not found")
	}

	return conn, nil
}

// DeleteConnection deletes a connection
func (s *ConnectionsService) DeleteConnection(ctx context.Context, tenantID, connectionID string) error {
	query := `DELETE FROM tenant_connections WHERE id = $1 AND tenant_id = $2`

	result, err := s.db.ExecContext(ctx, query, connectionID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("connection not found")
	}

	return nil
}

// LinkConnectionToDatasource links a connection to a datasource
func (s *ConnectionsService) LinkConnectionToDatasource(ctx context.Context, tenantID, datasourceID, connectionID string) error {
	query := `
		UPDATE tenant_product_datasource
		SET connection_id = $1, updated_at = NOW()
		WHERE id = $2 
		  AND tenant_product_id IN (
			SELECT id FROM tenant_product WHERE datasource_id IN (
				SELECT id FROM tenant_instance WHERE tenant_id = $3
			)
		  )
		  AND EXISTS (
			SELECT 1 FROM tenant_connections 
			WHERE id = $1 AND tenant_id = $3
		  )
	`

	result, err := s.db.ExecContext(ctx, query, connectionID, datasourceID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to link connection to datasource: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("datasource not found")
	}

	return nil
}

// UnlinkConnectionFromDatasource unlinks a connection from a datasource
func (s *ConnectionsService) UnlinkConnectionFromDatasource(ctx context.Context, tenantID, datasourceID string) error {
	query := `
		UPDATE tenant_product_datasource
		SET connection_id = NULL, updated_at = NOW()
		WHERE id = $1 AND tenant_product_id IN (
			SELECT id FROM tenant_product WHERE datasource_id IN (
				SELECT id FROM tenant_instance WHERE tenant_id = $2
			)
		)
	`

	result, err := s.db.ExecContext(ctx, query, datasourceID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to unlink connection from datasource: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("datasource not found")
	}

	return nil
}

// GetDatasourcesForConnection retrieves all datasources linked to a connection
func (s *ConnectionsService) GetDatasourcesForConnection(ctx context.Context, tenantID, connectionID string) ([]string, error) {
	query := `
		SELECT id FROM tenant_product_datasource
		WHERE connection_id = $1 AND tenant_product_id IN (
			SELECT id FROM tenant_product WHERE datasource_id IN (
				SELECT id FROM tenant_instance WHERE tenant_id = $2
			)
		)
	`

	rows, err := s.db.QueryContext(ctx, query, connectionID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get datasources: %w", err)
	}
	defer rows.Close()

	var datasourceIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan datasource id: %w", err)
		}
		datasourceIDs = append(datasourceIDs, id)
	}

	return datasourceIDs, rows.Err()
}
