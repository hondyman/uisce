package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/jmoiron/sqlx"
)

// Connection represents a datasource connection from tenant_connections table
type Connection struct {
	ID           string    `json:"id" db:"id"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	Name         string    `json:"name" db:"connection_name"`
	DatabaseType string    `json:"database_type" db:"database_type"`
	DSN          string    `json:"dsn" db:"dsn"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
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

	query := `
		INSERT INTO tenant_connections 
		(id, tenant_id, connection_name, database_type, dsn, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		conn.ID, conn.TenantID, conn.Name, conn.DatabaseType, conn.DSN,
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

	query := `
		SELECT id, tenant_id, connection_name, database_type, dsn, created_at, updated_at
		FROM tenant_connections
		WHERE id = $1 AND tenant_id = $2
	`

	err := s.db.QueryRowContext(ctx, query, connectionID, tenantID).Scan(
		&conn.ID, &conn.TenantID, &conn.Name, &conn.DatabaseType, &conn.DSN,
		&conn.CreatedAt, &conn.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connection not found")
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	return conn, nil
}

// ListConnections retrieves all connections for a tenant
func (s *ConnectionsService) ListConnections(ctx context.Context, tenantID string) ([]*Connection, error) {
	query := `
		SELECT id, tenant_id, connection_name, database_type, dsn, created_at, updated_at
		FROM tenant_connections
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return []*Connection{}, nil
	}
	defer rows.Close()

	var connections []*Connection
	for rows.Next() {
		conn := &Connection{}

		err := rows.Scan(
			&conn.ID, &conn.TenantID, &conn.Name, &conn.DatabaseType, &conn.DSN,
			&conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
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

	query := `
		UPDATE tenant_connections
		SET connection_name = $1, database_type = $2, dsn = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
	`

	result, err := s.db.ExecContext(ctx, query,
		conn.Name, conn.DatabaseType, conn.DSN,
		conn.UpdatedAt,
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

func (s *ConnectionsService) LinkConnectionToDatasource(ctx context.Context, tenantID, datasourceID, connectionID string) error {
	if err := db.RequireVerifiedTenantFromCtx(ctx); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	var result sql.Result
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
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
		var err2 error
		result, err2 = tx.ExecContext(ctx, query, connectionID, datasourceID, tenantID)
		return err2
	})
	if err != nil {
		return fmt.Errorf("failed to link connection to datasource: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("datasource not found")
	}
	return nil
}

func (s *ConnectionsService) UnlinkConnectionFromDatasource(ctx context.Context, tenantID, datasourceID string) error {
	if err := db.RequireVerifiedTenantFromCtx(ctx); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	var result sql.Result
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		query := `
			UPDATE tenant_product_datasource
			SET connection_id = NULL, updated_at = NOW()
			WHERE id = $1 AND tenant_product_id IN (
				SELECT id FROM tenant_product WHERE datasource_id IN (
					SELECT id FROM tenant_instance WHERE tenant_id = $2
				)
			)
		`
		var err2 error
		result, err2 = tx.ExecContext(ctx, query, datasourceID, tenantID)
		return err2
	})
	if err != nil {
		return fmt.Errorf("failed to unlink connection from datasource: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("datasource not found")
	}
	return nil
}

func (s *ConnectionsService) GetDatasourcesForConnection(ctx context.Context, tenantID, connectionID string) ([]string, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("security: tenant context required")
	}
	var datasourceIDs []string
	err := db.WithTenantTransaction(ctx, s.db.DB, tenantID, func(tx *sql.Tx) error {
		query := `
			SELECT id FROM tenant_product_datasource
			WHERE connection_id = $1 AND tenant_product_id IN (
				SELECT id FROM tenant_product WHERE datasource_id IN (
					SELECT id FROM tenant_instance WHERE tenant_id = $2
				)
			)
		`
		rows, err := tx.QueryContext(ctx, query, connectionID, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			datasourceIDs = append(datasourceIDs, id)
		}
		return rows.Err()
	})
	return datasourceIDs, err
}
